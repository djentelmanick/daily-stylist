import { useState } from 'react'
import { createItem, updateItem, type Item, type ItemFields, type Options } from './api'
import { forgetDraft, ItemForm } from './ItemForm'
import { useBackButton } from './telegram'
import { texts } from './texts'

const newItemDraft = 'draft:new-item'

export function AddItemScreen({
  options,
  onBack,
  onAdded,
}: {
  options: Options
  onBack: () => void
  onAdded: (item: Item) => void
}) {
  useBackButton(() => {
    forgetDraft(newItemDraft)
    onBack()
  })
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
      draftKey={newItemDraft}
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
  const draftKey = `draft:item:${item.id}`
  useBackButton(() => {
    forgetDraft(draftKey)
    onBack()
  })

  async function save(fields: ItemFields) {
    onSaved(await updateItem(item.id, fields))
  }

  return (
    <ItemForm
      options={options}
      title={texts.editItemTitle}
      submitText={texts.save}
      draftKey={draftKey}
      initial={item}
      onSubmit={save}
    />
  )
}
