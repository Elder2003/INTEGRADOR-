#!/usr/bin/env python3
"""
NextLab ML training pipeline.
Reads labeled exam sessions from PostgreSQL, trains an XGBoost classifier,
exports to ONNX, and registers the model in the ml_models table.

Usage:
  python3 train.py --db-dsn <dsn> --model-dir <path> [--min-sessions 50]
"""
import argparse
import json
import os
import sys
import uuid
from datetime import datetime, timezone

import numpy as np
import psycopg2
import psycopg2.extras

FEATURE_NAMES = [
    "tab_blurs_rate",
    "clipboard_rate",
    "heartbeat_gap_avg",
    "session_duration_min",
    "burst_windows",
    "events_per_minute",
]

LABEL_FRAUD = "academic_fraud"


def fetch_labeled_sessions(conn):
    with conn.cursor(cursor_factory=psycopg2.extras.DictCursor) as cur:
        cur.execute("""
            SELECT
                ar.temporal_metrics,
                CASE WHEN gd.override_reason = %s THEN 1 ELSE 0 END AS label
            FROM analysis_results ar
            JOIN exam_sessions es ON es.id = ar.session_id
            JOIN grade_decisions gd ON gd.session_id = es.id
            WHERE ar.analysis_type = 'heuristic'
              AND gd.override_reason IS NOT NULL
              AND ar.temporal_metrics IS NOT NULL
        """, (LABEL_FRAUD,))
        return cur.fetchall()


def extract_features(row):
    tm = row["temporal_metrics"]
    if isinstance(tm, str):
        tm = json.loads(tm)
    return [
        float(tm.get("events_per_minute", 0) * 60),       # tab_blurs_rate proxy
        float(tm.get("events_per_minute", 0)),             # clipboard_rate proxy
        0.0,                                               # heartbeat_gap_avg (not stored yet)
        float(tm.get("session_duration_min", 0)),
        float(tm.get("burst_windows", 0)),
        float(tm.get("events_per_minute", 0)),
    ]


def register_model(conn, version, file_path, session_count):
    with conn.cursor() as cur:
        cur.execute("UPDATE ml_models SET is_active = FALSE WHERE is_active = TRUE")
        cur.execute("""
            INSERT INTO ml_models
                (version, trained_at, training_session_count, file_path, feature_names, is_active)
            VALUES (%s, %s, %s, %s, %s, TRUE)
            RETURNING id
        """, (version, datetime.now(timezone.utc), session_count, file_path, FEATURE_NAMES))
        model_id = cur.fetchone()[0]
        conn.commit()
        return model_id


def main():
    parser = argparse.ArgumentParser(description="NextLab ML training pipeline")
    parser.add_argument("--db-dsn", required=True, help="PostgreSQL DSN")
    parser.add_argument("--model-dir", required=True, help="Directory to save .onnx files")
    parser.add_argument("--min-sessions", type=int, default=50, help="Minimum labeled sessions required")
    args = parser.parse_args()

    os.makedirs(args.model_dir, exist_ok=True)

    conn = psycopg2.connect(args.db_dsn)

    rows = fetch_labeled_sessions(conn)
    if len(rows) < args.min_sessions:
        print(f"ERROR: insufficient_training_data: need {args.min_sessions}, have {len(rows)}", file=sys.stderr)
        sys.exit(2)

    X = np.array([extract_features(r) for r in rows], dtype=np.float32)
    y = np.array([r["label"] for r in rows], dtype=np.int32)

    from sklearn.model_selection import train_test_split
    from xgboost import XGBClassifier
    from skl2onnx import convert_sklearn
    from skl2onnx.common.data_types import FloatTensorType

    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)

    clf = XGBClassifier(n_estimators=100, max_depth=4, use_label_encoder=False, eval_metric="logloss")
    clf.fit(X_train, y_train)

    accuracy = clf.score(X_test, y_test)
    print(f"Test accuracy: {accuracy:.4f}", flush=True)

    # Export to ONNX
    initial_type = [("float_input", FloatTensorType([None, len(FEATURE_NAMES)]))]
    onnx_model = convert_sklearn(clf, initial_types=initial_type)

    # Determine next version
    with conn.cursor() as cur:
        cur.execute("SELECT COALESCE(MAX(version), 0) + 1 FROM ml_models")
        version = cur.fetchone()[0]

    file_path = os.path.join(args.model_dir, f"model_v{version}.onnx")
    with open(file_path, "wb") as f:
        f.write(onnx_model.SerializeToString())

    print(f"Model saved to {file_path}", flush=True)

    model_id = register_model(conn, version, file_path, len(rows))
    print(f"Model registered: id={model_id} version={version}", flush=True)

    conn.close()


if __name__ == "__main__":
    main()
