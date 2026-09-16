import { ApiError, type ApiErrorCode } from './api'

export const texts = {
  loading: 'Загружаю…',
  openFromBot: 'Откройте приложение из бота в Telegram.',
  loadFailed: 'Не получилось загрузить гардероб. Закройте приложение и откройте заново.',

  greeting: (hour: number) =>
    hour < 5 ? 'Доброй ночи!' : hour < 12 ? 'Доброе утро!' : hour < 18 ? 'Добрый день!' : hour < 23 ? 'Добрый вечер!' : 'Доброй ночи!',
  today: (date: Date) => capitalize(date.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' })),
  wardrobe: 'Мой гардероб',
  wardrobeEmptyHint: 'Пока пусто',
  addItem: 'Добавить вещь',
  addItemHint: 'Название, цвета, сезоны',
  recommendation: 'Подобрать образ',
  recommendationHint: 'Из вашего гардероба по погоде',
  todayOutfit: 'Сегодня на вас',

  pickingOutfit: 'Смотрю погоду и подбираю образ…',
  cityNeeded: 'Чтобы подобрать образ по погоде, выберите город.',
  chooseCity: 'Выбрать город',
  changeCity: 'Сменить город',
  outfitNumber: (index: number, count: number) => `Вариант ${index} из ${count}`,
  anotherOutfit: 'Другой вариант',
  wear: 'Надеваю',
  worn: 'Надето',
  wornHint: 'Записал. В ближайшие дни постараюсь не повторять эту одежду.',

  cityTitle: 'Город',
  cityHint: 'Для него будет погода в подборе образа.',
  cityPlaceholder: 'Например, Казань',
  citySearching: 'Ищу…',
  cityNotFound: 'Ничего не нашлось. Проверьте название.',

  emptyWardrobe: 'В гардеробе пока ничего нет.',
  select: 'Выбрать',
  cancel: 'Отмена',
  selectHint: 'Удерживайте вещь, чтобы выбрать несколько',
  selectedCount: (count: number) => `Выбрано: ${count}`,
  deleteSelected: (count: number) => (count === 0 ? 'Выберите вещи' : `Удалить ${items(count)}`),
  deleting: 'Удаляю…',
  confirmDeleteOne: (name: string) => `Удалить «${name}»? Вернуть не получится.`,
  confirmDeleteMany: (count: number) => `Удалить ${items(count)}? Вернуть не получится.`,

  notInSeason: 'Не сезон',
  outOfSeasonNow: (season: string) => `Сейчас ${season.toLowerCase()}, вещь не по сезону`,
  status: 'Статус',
  colors: 'Цвета',
  yes: 'Да',
  no: 'Нет',
  edit: 'Изменить',
  delete: 'Удалить',

  newItemTitle: 'Новая вещь',
  editItemTitle: 'Изменить вещь',
  name: 'Название',
  namePlaceholder: 'Например, синее худи',
  description: 'Описание, необязательно',
  descriptionPlaceholder: 'Например, с капюшоном, из Uniqlo',
  category: 'Категория',
  mainColor: 'Основной цвет',
  extraColors: 'Дополнительные цвета',
  seasons: 'Сезоны',
  allSeasons: 'Все сезоны',
  warmthLevel: 'Насколько тёплая',
  warmthNotChosen: 'Не выбрано — сдвиньте ползунок',
  warmthLighter: 'Легче',
  warmthWarmer: 'Теплее',
  waterproof: 'Не промокает',
  add: 'Добавить',
  save: 'Сохранить',
  submitting: 'Сохраняю…',
  added: (name: string) => `Вещь «${name}» добавлена в гардероб`,
}

export function items(count: number): string {
  return `${count} ${plural(count, 'вещь', 'вещи', 'вещей')}`
}

function capitalize(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1)
}

function plural(count: number, one: string, few: string, many: string): string {
  const lastDigit = count % 10
  const lastTwoDigits = count % 100
  if (lastDigit === 1 && lastTwoDigits !== 11) {
    return one
  }
  if (lastDigit >= 2 && lastDigit <= 4 && (lastTwoDigits < 12 || lastTwoDigits > 14)) {
    return few
  }
  return many
}

export const errorTexts: Record<ApiErrorCode, string> = {
  unauthorized: 'Сессия устарела. Закройте приложение и откройте заново.',
  bad_request: 'Приложение отправило неверные данные. Обновите его.',
  invalid_item: 'Проверьте, что все поля заполнены правильно.',
  not_found: 'Вещь не найдена: возможно, её уже удалили.',
  wardrobe_full: 'Гардероб полон: удалите что-нибудь из архива.',
  invalid_location: 'Не получилось сохранить этот город. Выберите другой.',
  location_not_set: 'Сначала выберите город.',
  weather_unavailable: 'Сервис погоды не отвечает. Попробуйте позже.',
  internal: 'Что-то пошло не так. Попробуйте позже.',
  network: 'Нет связи с сервером. Проверьте интернет.',
}

export function errorText(error: unknown): string {
  return errorTexts[error instanceof ApiError ? error.code : 'internal']
}
