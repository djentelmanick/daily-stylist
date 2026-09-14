import type { Item } from './api'
import { items, texts } from './texts'
import { Swatch } from './ui'

export function HomeScreen({
  itemCount,
  todayItems,
  onOpenWardrobe,
  onAddItem,
  onRecommend,
}: {
  itemCount: number
  todayItems: Item[]
  onOpenWardrobe: () => void
  onAddItem: () => void
  onRecommend: () => void
}) {
  const now = new Date()

  return (
    <div className="screen">
      <header className="home-header">
        <p className="hint">{texts.today(now)}</p>
        <h1>{texts.greeting(now.getHours())}</h1>
      </header>

      {todayItems.length > 0 && (
        <section className="today-card">
          <p className="hint">{texts.todayOutfit}</p>
          <ul className="today-items">
            {todayItems.map((item) => (
              <li key={item.id} className="today-item">
                <Swatch color={item.main_color} />
                {item.name}
              </li>
            ))}
          </ul>
        </section>
      )}

      <button type="button" className="hero" onClick={onRecommend}>
        <span className="hero-icon">
          <Icon path={icons.sparkles} />
        </span>
        <span className="hero-text">
          <span className="hero-title">{texts.recommendation}</span>
          <span className="hero-hint">{texts.recommendationHint}</span>
        </span>
        <Icon path={icons.chevron} />
      </button>

      <div className="tiles">
        <Tile
          icon={icons.hanger}
          title={texts.wardrobe}
          hint={itemCount === 0 ? texts.wardrobeEmptyHint : items(itemCount)}
          onClick={onOpenWardrobe}
        />
        <Tile icon={icons.plus} title={texts.addItem} hint={texts.addItemHint} onClick={onAddItem} />
      </div>
    </div>
  )
}

function Tile({ icon, title, hint, onClick }: { icon: string; title: string; hint: string; onClick: () => void }) {
  return (
    <button type="button" className="tile" onClick={onClick}>
      <span className="tile-icon">
        <Icon path={icon} />
      </span>
      <span className="tile-title">{title}</span>
      <span className="hint">{hint}</span>
    </button>
  )
}

const icons = {
  sparkles: 'M12 3l1.9 5.1L19 10l-5.1 1.9L12 17l-1.9-5.1L5 10l5.1-1.9zM19 15l.8 2.2 2.2.8-2.2.8L19 21l-.8-2.2-2.2-.8 2.2-.8z',
  hanger: 'M10 5.5a2 2 0 1 1 2.8 1.8c-.5.2-.8.7-.8 1.2v1L3.4 15.6A1.3 1.3 0 0 0 4.2 18h15.6a1.3 1.3 0 0 0 .8-2.4L12 9.5',
  plus: 'M12 5v14M5 12h14',
  chevron: 'M9 6l6 6-6 6',
}

function Icon({ path }: { path: string }) {
  return (
    <svg
      className="icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={path} />
    </svg>
  )
}
