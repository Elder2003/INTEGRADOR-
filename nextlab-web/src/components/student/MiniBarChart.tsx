interface Props {
  data: number[]
  maxValue?: number
}

export default function MiniBarChart({ data, maxValue }: Props) {
  const max = maxValue ?? Math.max(...data, 1)

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'flex-end',
        gap: 2,
        height: 40,
        padding: '0 2px',
      }}
    >
      {data.map((val, i) => (
        <div
          key={i}
          title={`${val} ppm`}
          style={{
            flex: 1,
            height: `${Math.max(8, (val / max) * 100)}%`,
            backgroundColor: 'var(--color-accent)',
            opacity: 0.5 + (i / data.length) * 0.5,
            borderRadius: '2px 2px 0 0',
            transition: 'height 0.4s ease',
          }}
        />
      ))}
    </div>
  )
}
