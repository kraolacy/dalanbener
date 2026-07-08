import PostCard from './PostCard.jsx'

export default function Feed({ posts, onOpen }) {
  if (!posts.length) {
    return (
      <div className="empty">
        <div className="big">🫥</div>
        这个圈子还没有帖子，来发第一篇散帅动态吧！
      </div>
    )
  }
  return (
    <div className="feed">
      {posts.map((p) => (
        <PostCard key={p.id} post={p} onOpen={onOpen} />
      ))}
    </div>
  )
}
