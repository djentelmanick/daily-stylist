import type { ApiErrorCode } from './api'

export const texts = {
  title: 'Новая вещь',
  loading: 'Загружаю…',
  openFromBot: 'Откройте приложение из бота в Telegram.',
  optionsFailed: 'Не получилось загрузить форму. Закройте приложение и откройте заново.',
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
  submit: 'Добавить',
  submitting: 'Сохраняю…',
  added: (name: string) => `Вещь «${name}» добавлена в гардероб`,
}

export const errorTexts: Record<ApiErrorCode, string> = {
  unauthorized: 'Сессия устарела. Закройте приложение и откройте заново.',
  bad_request: 'Приложение отправило неверные данные. Обновите его.',
  invalid_item: 'Проверьте, что все поля заполнены правильно.',
  wardrobe_full: 'Гардероб полон: удалите что-нибудь из архива.',
  internal: 'Не получилось сохранить. Попробуйте позже.',
  network: 'Нет связи с сервером. Проверьте интернет.',
}
