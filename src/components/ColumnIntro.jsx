import { daysUntil } from '../data.js'

// 选中带 blurb 的圈子时，在信息流上方显示一张专栏介绍条
export default function ColumnIntro({ cat }) {
  if (!cat || !cat.blurb) return null
  const days = cat.countdownDate ? daysUntil(cat.countdownDate.m, cat.countdownDate.d) : null
  return (
    <div
      className="column-intro"
      style={{ background: `linear-gradient(135deg, ${cat.grad[0]}, ${cat.grad[1]})` }}
    >
      <div className="ci-emoji">{cat.emoji}</div>
      <div className="ci-text">
        <h3>{cat.name}</h3>
        <p>{cat.blurb}</p>
        {days !== null && (
          <span className="ci-count">
            {days === 0
              ? `🎉 就是今天！${cat.name}快乐`
              : `⏳ 距 ${cat.countdownDate.m}·${cat.countdownDate.d} ${cat.name}还有 ${days} 天`}
          </span>
        )}
      </div>
    </div>
  )
}
