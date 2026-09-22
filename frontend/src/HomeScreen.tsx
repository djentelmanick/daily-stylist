import type { Item } from './api'
import { items, texts } from './texts'
import { Chevron, Icon, ItemAvatar } from './ui'

export function HomeScreen({
  itemCount,
  todayItems,
  onOpenItem,
  onOpenToday,
  onOpenWardrobe,
  onAddItem,
  onRecommend,
  onOpenSettings,
}: {
  itemCount: number
  todayItems: Item[]
  onOpenItem: (itemId: number) => void
  onOpenToday: () => void
  onOpenWardrobe: () => void
  onAddItem: () => void
  onRecommend: () => void
  onOpenSettings: () => void
}) {
  const now = new Date()

  return (
    <div className="screen">
      <header className="home-header">
        <div>
          <p className="hint">{texts.today(now)}</p>
          <h1>{texts.greeting(now.getHours())}</h1>
        </div>
        <button type="button" className="icon-button" aria-label={texts.settings} onClick={onOpenSettings}>
          <Icon path={icons.gear} />
        </button>
      </header>

      {todayItems.length === 0 && itemCount > 0 && (
        <section className="today-card">
          <div className="card-row">
            <p className="hint">{texts.todayOutfitEmpty}</p>
            <button type="button" className="link-button" onClick={onOpenToday}>
              {texts.assembleOutfit}
            </button>
          </div>
        </section>
      )}
      {todayItems.length > 0 && (
        <section className="today-card">
          <div className="card-row">
            <p className="hint">{texts.todayOutfit}</p>
            <button type="button" className="link-button" onClick={onOpenToday}>
              {texts.edit}
            </button>
          </div>
          <ul className="today-items">
            {todayItems.map((item) => (
              <li key={item.id}>
                <button type="button" className="today-item" onClick={() => onOpenItem(item.id)}>
                  <ItemAvatar item={item} />
                  {item.name}
                </button>
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
        <Chevron direction="right" />
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
  gear:
    'M12 15.5a3.5 3.5 0 1 1 0-7 3.5 3.5 0 0 1 0 7zM19.4 13.4a7.7 7.7 0 0 0 0-2.8l2-1.4-2-3.4-2.3 1a7.7 7.7 0 0 0-2.4-1.4L14.4 3h-4l-.3 2.4a7.7 7.7 0 0 0-2.4 1.4l-2.3-1-2 3.4 2 1.4a7.7 7.7 0 0 0 0 2.8l-2 1.4 2 3.4 2.3-1a7.7 7.7 0 0 0 2.4 1.4l.3 2.4h4l.3-2.4a7.7 7.7 0 0 0 2.4-1.4l2.3 1 2-3.4z',
}
