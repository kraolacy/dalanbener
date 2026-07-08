import { useEffect, useState } from 'react'
import Feed from './Feed.jsx'

// 计算距离下一个 8 月 3 日的倒计时
function nextSunshineDay() {
  const now = new Date()
  let year = now.getFullYear()
  let target = new Date(year, 7, 3, 0, 0, 0) // 月份 0-based，7 = 八月
  if (now > target) target = new Date(year + 1, 7, 3, 0, 0, 0)
  return target
}

function useCountdown() {
  const [left, setLeft] = useState(() => nextSunshineDay() - new Date())
  useEffect(() => {
    const t = setInterval(() => setLeft(nextSunshineDay() - new Date()), 1000)
    return () => clearInterval(t)
  }, [])
  const s = Math.max(0, Math.floor(left / 1000))
  return {
    days: Math.floor(s / 86400),
    hours: Math.floor((s % 86400) / 3600),
    mins: Math.floor((s % 3600) / 60),
    secs: s % 60,
    isToday: new Date().getMonth() === 7 && new Date().getDate() === 3,
  }
}

export default function Festival({ posts, onOpen }) {
  const cd = useCountdown()
  const festivalPosts = posts.filter((p) => p.festival)

  return (
    <div>
      <section className="festival-hero">
        <span className="sun-emoji">🌞</span>
        <div className="en">International Sunshine Day · Aug 3</div>
        <h1>8·3 国际散帅节</h1>
        <p>
          三月八号是女神节，八月三号是散帅节。
          今天，放下手机，散步、散心、散个帅——做一个阳光、松弛、爱交朋友的男人。
        </p>

        {cd.isToday ? (
          <div className="countdown">
            <div className="count-box" style={{ minWidth: 'auto', padding: '10px 18px' }}>
              <b style={{ fontSize: 18 }}>🎉 散帅节快乐！</b>
              <span>今天就是你的节日</span>
            </div>
          </div>
        ) : (
          <div className="countdown">
            <div className="count-box"><b>{cd.days}</b><span>天</span></div>
            <div className="count-box"><b>{String(cd.hours).padStart(2, '0')}</b><span>时</span></div>
            <div className="count-box"><b>{String(cd.mins).padStart(2, '0')}</b><span>分</span></div>
            <div className="count-box"><b>{String(cd.secs).padStart(2, '0')}</b><span>秒</span></div>
          </div>
        )}
      </section>

      <div className="page" style={{ paddingBottom: 0 }}>
        <div>
          <span className="festival-tag">#我的散帅时刻</span>
          <span className="festival-tag">#散帅一夏</span>
          <span className="festival-tag">#约个散帅局</span>
          <span className="festival-tag">#阳光男孩</span>
        </div>
        <div className="section-title">🌞 散帅节精选</div>
      </div>

      <Feed posts={festivalPosts} onOpen={onOpen} />
    </div>
  )
}
