import { useState } from 'react'
import { createItem, updateItem, type Item, type ItemFields, type Options } from './api'
import { ItemForm } from './ItemForm'
import { useBackButton } from './telegram'
import { texts } from './texts'

export function AddItemScreen({
  options,
  onBack,
  onAdded,
}: {
  options: Options
  onBack: () => void
  onAdded: (item: Item) => void
}) {
  useBackButton(onBack)
  const [formKey, setFormKey] = useState(0)
  const [addedName, setAddedName] = useState('')

  async function add(fields: ItemFields) {
    setAddedName('')
    const item = await createItem(fields)
    onAdded(item)
    setAddedName(item.name)
    setFormKey((current) => current + 1)
  }

  return (
    <ItemForm
      key={formKey}
      options={options}
      title={texts.newItemTitle}
      submitText={texts.add}
      notice={addedName === '' ? undefined : texts.added(addedName)}
      onSubmit={add}
    />
  )
}

export function EditItemScreen({
  options,
  item,
  onBack,
  onSaved,
}: {
  options: Options
  item: Item
  onBack: () => void
  onSaved: (item: Item) => void
}) {
  useBackButton(onBack)

  async function save(fields: ItemFields) {
    onSaved(await updateItem(item.id, fields))
  }

  return (
    <ItemForm options={options} title={texts.editItemTitle} submitText={texts.save} initial={item} onSubmit={save} />
  )
}
