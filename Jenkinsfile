pipeline {
  agent { label 'nextlab-ci' }

  options {
    skipDefaultCheckout(true)
    disableConcurrentBuilds()
    timeout(time: 45, unit: 'MINUTES')
    buildDiscarder(logRotator(numToKeepStr: '30', artifactNumToKeepStr: '15'))
  }

  environment {
    CI = 'true'
    GOBIN = "${WORKSPACE}/.ci-tools/bin"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
        sh 'mkdir -p reports'
      }
    }

    stage('Verificar agente') {
      steps {
        sh '''#!/usr/bin/env bash
          set -euo pipefail
          docker version
          docker compose version
          go version
          node --version
          corepack --version
          curl --version
          jq --version
        '''
      }
    }

    stage('Calidad frontend') {
      steps {
        dir('nextlab-web') {
          sh '''#!/usr/bin/env bash
            set -euo pipefail
            corepack enable
            corepack pnpm install --frozen-lockfile
            corepack pnpm lint
            corepack pnpm build
          '''
        }
      }
    }

    stage('Pruebas unitarias Go') {
      steps {
        dir('nextlab-api') {
          sh '''#!/usr/bin/env bash
            set -uo pipefail
            mkdir -p "$GOBIN" ../reports
            go install github.com/jstemmer/go-junit-report/v2@v2.1.0

            set +e
            go test -v ./... > ../reports/go-unit.log 2>&1
            test_status=$?
            set -e

            "$GOBIN/go-junit-report" < ../reports/go-unit.log > ../reports/go-unit.xml
            exit "$test_status"
          '''
        }
      }
    }

    stage('Prueba integral aislada') {
      steps {
        dir('nextlab-api') {
          sh '''#!/usr/bin/env bash
            set -uo pipefail
            set +e
            ./scripts/test-system.sh > ../reports/system-test.log 2>&1
            test_status=$?
            set -e

            if [ "$test_status" -eq 0 ]; then
              printf '%s\n' '<?xml version="1.0" encoding="UTF-8"?>' \
                '<testsuite name="nextlab-system" tests="1" failures="0" errors="0">' \
                '  <testcase classname="nextlab.system" name="end-to-end"/>' \
                '</testsuite>' > ../reports/system-test.xml
            else
              printf '%s\n' '<?xml version="1.0" encoding="UTF-8"?>' \
                '<testsuite name="nextlab-system" tests="1" failures="1" errors="0">' \
                "  <testcase classname=\"nextlab.system\" name=\"end-to-end\"><failure message=\"El script test-system.sh terminó con código $test_status. Revise system-test.log.\"/></testcase>" \
                '</testsuite>' > ../reports/system-test.xml
            fi
            exit "$test_status"
          '''
        }
      }
    }
  }

  post {
    always {
      junit allowEmptyResults: true, keepLongStdio: true, testResults: 'reports/**/*.xml'
      archiveArtifacts allowEmptyArchive: true, artifacts: 'reports/**,nextlab-web/dist/**', fingerprint: true
    }
    cleanup {
      deleteDir()
    }
  }
}
