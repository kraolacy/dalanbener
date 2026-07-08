import { catByKey } from '../data.js'
import { useStore } from '../store.jsx'

export default function PostCard({ post, onOpen }) {
  const { toggleLike } = useStore()
  const cat = catByKey(post.cat)
  const emoji = post.cover || cat.emoji
  const grad = post.festival ? ['#ffb200', '#ff7a3d'] : cat.grad
  const tag = post.festival ? '🌞 散帅节' : cat.festive ? `${cat.emoji} ${cat.name}` : null

  return (
    <article className="card" onClick={() => onOpen(post)}>
      <div
        className="cover"
        style={{
          background: `linear-gradient(135deg, ${grad[0]}, ${grad[1]})`,
          minHeight: post.tall ? 150 : 108,
        }}
      >
        {tag && <span className="cover-tag">{tag}</span>}
        <span className="emoji">{emoji}</span>
      </div>
      <div className="card-body">
        <div className="card-title">{post.title}</div>
        <div className="card-meta">
          <div className="author">
            <span className="avatar">{post.avatar}</span>
            <span>{post.author}</span>
          </div>
          <button
            className={`like-btn ${post.liked ? 'on' : ''}`}
            onClick={(e) => { e.stopPropagation(); toggleLike(post.id) }}
            aria-label="点赞"
          >
            <span className="heart">{post.liked ? '❤️' : '🤍'}</span>
            {post.likeCount}
          </button>
        </div>
      </div>
    </article>
  )
}
