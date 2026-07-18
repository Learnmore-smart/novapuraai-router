import { zodResolver } from '@hookform/resolvers/zod'
import { Braces, Plus, Save, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { StaticRowActions } from '@/components/data-table/static/static-row-actions'
import { Dialog } from '@/components/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

import { SettingsSwitchField } from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  FAQ_TRANSLATION_LANGUAGES,
  getFAQEntryTranslation,
  parseFAQBatch,
  type FAQEntry,
  type FAQTranslationLanguage,
} from './faq-json'

type FAQ = FAQEntry
type FAQFilterLanguage = 'all' | FAQTranslationLanguage

type FAQSectionProps = {
  enabled: boolean
  data: string
}

const faqSchema = z.object({
  question: z
    .string()
    .min(1, 'Question is required')
    .max(200, 'Question must be less than 200 characters'),
  answer: z
    .string()
    .min(1, 'Answer is required')
    .max(1000, 'Answer must be less than 1000 characters'),
})

type FAQFormValues = z.infer<typeof faqSchema>

const FAQ_FORM_ID = 'faq-form'
const FAQ_JSON_EXAMPLE = JSON.stringify(
  [
    {
      translations: {
        en: {
          question: 'Do registrations receive credits?',
          answer: 'Check the current welcome offer after registration.',
        },
        zh: {
          question: '注册会赠送额度吗？',
          answer: '注册后请查看当前的新用户活动说明。',
        },
        'zh-TW': {
          question: '註冊會贈送額度嗎？',
          answer: '註冊後請查看目前的新用戶活動說明。',
        },
        fr: {
          question: 'L’inscription donne-t-elle des crédits ?',
          answer: 'Consultez l’offre de bienvenue actuelle après votre inscription.',
        },
        ja: {
          question: '登録するとクレジットは付与されますか？',
          answer: '登録後に現在の新規登録特典をご確認ください。',
        },
        ru: {
          question: 'Начисляются ли кредиты за регистрацию?',
          answer: 'После регистрации ознакомьтесь с текущим приветственным предложением.',
        },
        vi: {
          question: 'Đăng ký có được tặng hạn mức không?',
          answer: 'Sau khi đăng ký, hãy xem ưu đãi chào mừng hiện tại.',
        },
      },
    },
  ],
  null,
  2
)

const FAQ_LANGUAGE_FILTERS: { value: FAQFilterLanguage; label: string }[] = [
  { value: 'all', label: 'All' },
  ...FAQ_TRANSLATION_LANGUAGES.map((language) => ({
    value: language,
    label: language.toUpperCase(),
  })),
]

export function FAQSection({ enabled, data }: FAQSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [faqList, setFaqList] = useState<FAQ[]>([])
  const [isEnabled, setIsEnabled] = useState(enabled)
  const [hasChanges, setHasChanges] = useState(false)
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [languageFilter, setLanguageFilter] = useState<FAQFilterLanguage>('all')
  const [showDialog, setShowDialog] = useState(false)
  const [showJSONDialog, setShowJSONDialog] = useState(false)
  const [faqJSON, setFaqJSON] = useState(FAQ_JSON_EXAMPLE)
  const [faqJSONError, setFaqJSONError] = useState('')
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [editingFaq, setEditingFaq] = useState<FAQ | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<'single' | 'batch'>('single')

  const form = useForm<FAQFormValues>({
    resolver: zodResolver(faqSchema),
    defaultValues: {
      question: '',
      answer: '',
    },
  })

  useEffect(() => {
    try {
      const parsed = JSON.parse(data || '[]')
      if (Array.isArray(parsed)) {
        setFaqList(
          parsed.map((item, idx) => ({
            ...item,
            id: item.id || idx + 1,
          }))
        )
      }
    } catch {
      setFaqList([])
    }
  }, [data])

  useEffect(() => {
    setIsEnabled(enabled)
  }, [enabled])

  const handleToggleEnabled = async (checked: boolean) => {
    try {
      await updateOption.mutateAsync({
        key: 'console_setting.faq_enabled',
        value: checked,
      })
      setIsEnabled(checked)
      toast.success(t('Setting saved'))
    } catch {
      toast.error(t('Failed to update setting'))
    }
  }

  const handleAdd = () => {
    setEditingFaq(null)
    form.reset({
      question: '',
      answer: '',
    })
    setShowDialog(true)
  }

  const handleEdit = (faq: FAQ) => {
    setEditingFaq(faq)
    const localized =
      languageFilter === 'all'
        ? { question: faq.question, answer: faq.answer }
        : getFAQEntryTranslation(faq, languageFilter)
    form.reset({
      question: localized?.question ?? faq.question,
      answer: localized?.answer ?? faq.answer,
    })
    setShowDialog(true)
  }

  const handleDelete = (faq: FAQ) => {
    setEditingFaq(faq)
    setDeleteTarget('single')
    setShowDeleteDialog(true)
  }

  const handleBatchDelete = () => {
    if (selectedIds.length === 0) {
      toast.error(t('Please select items to delete'))
      return
    }
    setDeleteTarget('batch')
    setShowDeleteDialog(true)
  }

  const confirmDelete = () => {
    if (deleteTarget === 'single' && editingFaq) {
      setFaqList((prev) => prev.filter((item) => item.id !== editingFaq.id))
      setHasChanges(true)
      toast.success(t('FAQ deleted. Click "Save Settings" to apply.'))
    } else if (deleteTarget === 'batch') {
      setFaqList((prev) =>
        prev.filter((item) => !selectedIds.includes(item.id))
      )
      setSelectedIds([])
      setHasChanges(true)
      toast.success(
        t('{{count}} FAQs deleted. Click "Save Settings" to apply.', {
          count: selectedIds.length,
        })
      )
    }
    setShowDeleteDialog(false)
    setEditingFaq(null)
  }

  const handleSubmitForm = (values: FAQFormValues) => {
    if (editingFaq) {
      setFaqList((prev) =>
        prev.map((item) => {
          if (item.id !== editingFaq.id || !item.translations) {
            return item.id === editingFaq.id ? { ...item, ...values } : item
          }

          const editedLanguage =
            languageFilter === 'all'
              ? FAQ_TRANSLATION_LANGUAGES.find(
                  (language) => item.translations?.[language]
                )
              : languageFilter
          if (!editedLanguage) return { ...item, ...values }

          const translations = {
            ...item.translations,
            [editedLanguage]: values,
          }
          const fallback =
            translations.en ?? translations[editedLanguage] ?? values
          return {
            ...item,
            question: fallback.question,
            answer: fallback.answer,
            translations,
          }
        })
      )
      toast.success(t('FAQ updated. Click "Save Settings" to apply.'))
    } else {
      const newId = Math.max(...faqList.map((item) => item.id), 0) + 1
      setFaqList((prev) => [...prev, { id: newId, ...values }])
      toast.success(t('FAQ added. Click "Save Settings" to apply.'))
    }
    setHasChanges(true)
    setShowDialog(false)
  }

  const handleOpenJSONDialog = () => {
    setFaqJSON(FAQ_JSON_EXAMPLE)
    setFaqJSONError('')
    setShowJSONDialog(true)
  }

  const handleImportJSON = () => {
    const result = parseFAQBatch(
      faqJSON,
      faqList.map((item) => item.id)
    )
    if (!result.success) {
      setFaqJSONError(t(result.error, result.values))
      return
    }

    setFaqList((previous) => [...previous, ...result.entries])
    setHasChanges(true)
    setShowJSONDialog(false)
    setFaqJSONError('')
    toast.success(
      t('{{count}} FAQ entries imported. Click "Save Settings" to apply.', {
        count: result.entries.length,
      })
    )
  }

  const handleSaveAll = async () => {
    try {
      await updateOption.mutateAsync({
        key: 'console_setting.faq',
        value: JSON.stringify(faqList),
      })
      setHasChanges(false)
      toast.success(t('FAQ saved successfully'))
    } catch {
      toast.error(t('Failed to save FAQ'))
    }
  }

  const toggleSelectOne = (id: number, checked: boolean) => {
    setSelectedIds((prev) =>
      checked ? [...prev, id] : prev.filter((item) => item !== id)
    )
  }

  const visibleFAQs =
    languageFilter === 'all'
      ? faqList
      : faqList.filter((faq) =>
          Boolean(getFAQEntryTranslation(faq, languageFilter))
        )

  return (
    <SettingsSection title={t('FAQ')}>
      <div className='space-y-4'>
        <div className='flex flex-wrap items-center justify-between gap-2'>
          <div className='flex flex-wrap items-center gap-2'>
            <Button onClick={handleAdd} size='sm'>
              <Plus className='mr-2 h-4 w-4' />
              {t('Add FAQ')}
            </Button>
            <Button
              type='button'
              onClick={handleOpenJSONDialog}
              size='sm'
              variant='outline'
            >
              <Braces className='mr-2 h-4 w-4' aria-hidden='true' />
              {t('Import JSON')}
            </Button>
            <Button
              onClick={handleBatchDelete}
              size='sm'
              variant='destructive'
              disabled={selectedIds.length === 0}
            >
              <Trash2 className='mr-2 h-4 w-4' />
              {t('Delete (')}
              {selectedIds.length})
            </Button>
            <Button
              onClick={handleSaveAll}
              size='sm'
              variant='secondary'
              disabled={!hasChanges || updateOption.isPending}
            >
              <Save className='mr-2 h-4 w-4' />
              {updateOption.isPending ? t('Saving...') : t('Save Settings')}
            </Button>
          </div>
          <div className='flex flex-wrap items-center gap-1' role='group'>
            {FAQ_LANGUAGE_FILTERS.map((filter) => (
              <Button
                key={filter.value}
                type='button'
                size='sm'
                variant={
                  languageFilter === filter.value ? 'secondary' : 'outline'
                }
                onClick={() => {
                  setLanguageFilter(filter.value)
                  setSelectedIds([])
                }}
              >
                {filter.value === 'all' ? t(filter.label) : filter.label}
              </Button>
            ))}
          </div>
          <SettingsSwitchField
            checked={isEnabled}
            onCheckedChange={handleToggleEnabled}
            label={t('Enabled')}
            className='py-0'
          />
        </div>

        <StaticDataTable
          data={visibleFAQs}
          getRowKey={(faq) => faq.id}
          emptyContent={t('No FAQ entries yet. Click "Add FAQ" to create one.')}
          columns={[
            {
              id: 'select',
              header: (
                <Checkbox
                  checked={
                    selectedIds.length === visibleFAQs.length &&
                    visibleFAQs.length > 0
                  }
                  onCheckedChange={(checked) =>
                    setSelectedIds(
                      checked ? visibleFAQs.map((item) => item.id) : []
                    )
                  }
                />
              ),
              className: 'w-12',
              cell: (faq) => (
                <Checkbox
                  checked={selectedIds.includes(faq.id)}
                  onCheckedChange={(checked) =>
                    toggleSelectOne(faq.id, checked as boolean)
                  }
                />
              ),
            },
            {
              id: 'question',
              header: t('Question'),
              cellClassName: 'max-w-xs truncate font-medium',
              cell: (faq) =>
                languageFilter === 'all'
                  ? faq.question
                  : getFAQEntryTranslation(faq, languageFilter)?.question ?? '',
            },
            {
              id: 'answer',
              header: t('Answer'),
              cellClassName: 'text-muted-foreground max-w-md truncate',
              cell: (faq) =>
                languageFilter === 'all'
                  ? faq.answer
                  : getFAQEntryTranslation(faq, languageFilter)?.answer ?? '',
            },
            {
              id: 'actions',
              header: t('Actions'),
              cell: (faq) => (
                <StaticRowActions
                  editLabel={t('Edit')}
                  deleteLabel={t('Delete')}
                  menuLabel={t('Open menu')}
                  onEdit={() => handleEdit(faq)}
                  onDelete={() => handleDelete(faq)}
                />
              ),
            },
          ]}
        />
      </div>

      <Dialog
        open={showDialog}
        onOpenChange={setShowDialog}
        title={editingFaq ? t('Edit FAQ') : t('Add FAQ')}
        description={t('Create or update frequently asked questions for users')}
        contentClassName='max-w-2xl'
        contentHeight='auto'
        bodyClassName='space-y-4'
        footer={
          <>
            <Button
              type='button'
              variant='outline'
              onClick={() => setShowDialog(false)}
            >
              {t('Cancel')}
            </Button>
            <Button type='submit' form={FAQ_FORM_ID}>
              {editingFaq ? t('Update') : t('Add')}
            </Button>
          </>
        }
      >
        <Form {...form}>
          <form
            id={FAQ_FORM_ID}
            onSubmit={form.handleSubmit(handleSubmitForm)}
            className='space-y-4'
          >
            <FormField
              control={form.control}
              name='question'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Question')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('How to reset my quota?')}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Maximum 200 characters')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='answer'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Answer')}</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder={t(
                        'Visit Settings → General and adjust quota options...'
                      )}
                      rows={8}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('Maximum 1000 characters. Supports Markdown and HTML.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </form>
        </Form>
      </Dialog>

      <Dialog
        open={showJSONDialog}
        onOpenChange={setShowJSONDialog}
        title={t('Import FAQ JSON')}
        description={t(
          'Paste a JSON array of questions and answers. Imported entries are added to the current draft.'
        )}
        contentClassName='max-w-3xl'
        contentHeight='auto'
        bodyClassName='space-y-3'
        footer={
          <>
            <Button
              type='button'
              variant='outline'
              onClick={() => setShowJSONDialog(false)}
            >
              {t('Cancel')}
            </Button>
            <Button type='button' onClick={handleImportJSON}>
              <Braces aria-hidden='true' />
              {t('Import entries')}
            </Button>
          </>
        }
      >
        <div className='space-y-2'>
          <label className='text-sm font-medium' htmlFor='faq-json-import'>
            {t('FAQ JSON')}
          </label>
          <Textarea
            id='faq-json-import'
            value={faqJSON}
            onChange={(event) => {
              setFaqJSON(event.target.value)
              setFaqJSONError('')
            }}
            rows={16}
            spellCheck={false}
            className='font-mono text-xs'
            aria-invalid={faqJSONError !== ''}
            aria-describedby={faqJSONError ? 'faq-json-error' : undefined}
          />
          {faqJSONError && (
            <p id='faq-json-error' className='text-destructive text-sm'>
              {faqJSONError}
            </p>
          )}
        </div>
      </Dialog>

      <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Are you sure?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteTarget === 'single'
                ? t('This FAQ entry will be removed from the list.')
                : t('{{count}} FAQ entries will be removed from the list.', {
                    count: selectedIds.length,
                  })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction variant='destructive' onClick={confirmDelete}>
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}
