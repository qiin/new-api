/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import { CopyButton } from '@/components/copy-button'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { removeTrailingSlash } from './utils'

const createPayerScanSchema = (t: (key: string) => string) =>
  z.object({
    PayerScanEnabled: z.boolean(),
    PayerScanMerchantID: z.string(),
    PayerScanApiKey: z.string(),
    PayerScanBaseURL: z.string().refine((value) => {
      const trimmed = value.trim()
      if (!trimmed) return true
      return /^https?:\/\//.test(trimmed)
    }, t('Provide a valid URL starting with http:// or https://')),
    PayerScanUnitPrice: z.number().positive(),
    PayerScanMinTopUp: z.number().int().positive(),
  })

export type PayerScanSettingsValues = z.infer<
  ReturnType<typeof createPayerScanSchema>
>

type PayerScanSettingsSectionProps = {
  defaultValues: PayerScanSettingsValues
  /** Public callback endpoint the operator registers in their PayerScan store */
  webhookUrl: string
  complianceConfirmed: boolean
  /** True once a merchant id is stored, so the key field can stay blank */
  hasStoredApiKey: boolean
}

export function PayerScanSettingsSection(props: PayerScanSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const form = useForm<PayerScanSettingsValues>({
    resolver: zodResolver(createPayerScanSchema(t)),
    defaultValues: props.defaultValues,
  })

  useResetForm(form, props.defaultValues)

  const onSubmit = async (values: PayerScanSettingsValues) => {
    const merchantId = values.PayerScanMerchantID.trim()
    const apiKey = values.PayerScanApiKey.trim()
    const baseUrl = removeTrailingSlash(values.PayerScanBaseURL.trim())

    const updates: Array<{ key: string; value: string | boolean | number }> = []

    if (merchantId !== props.defaultValues.PayerScanMerchantID.trim()) {
      updates.push({ key: 'PayerScanMerchantID', value: merchantId })
    }
    // The stored key is never sent back to the console, so an empty field
    // means "keep the existing key" rather than "clear it".
    if (apiKey) {
      updates.push({ key: 'PayerScanApiKey', value: apiKey })
    }
    if (baseUrl !== removeTrailingSlash(props.defaultValues.PayerScanBaseURL)) {
      updates.push({ key: 'PayerScanBaseURL', value: baseUrl })
    }
    if (values.PayerScanUnitPrice !== props.defaultValues.PayerScanUnitPrice) {
      updates.push({
        key: 'PayerScanUnitPrice',
        value: values.PayerScanUnitPrice,
      })
    }
    if (values.PayerScanMinTopUp !== props.defaultValues.PayerScanMinTopUp) {
      updates.push({
        key: 'PayerScanMinTopUp',
        value: values.PayerScanMinTopUp,
      })
    }
    // Sent last so the gateway only turns on once its credentials are stored.
    if (values.PayerScanEnabled !== props.defaultValues.PayerScanEnabled) {
      updates.push({ key: 'PayerScanEnabled', value: values.PayerScanEnabled })
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  return (
    <SettingsSection title={t('PayerScan (Crypto)')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save PayerScan settings'
          />

          {!props.complianceConfirmed && (
            <Alert>
              <AlertDescription>
                {t(
                  'Payment, redemption codes, subscription plans, and invitation rewards are locked until the root administrator confirms the compliance terms.'
                )}
              </AlertDescription>
            </Alert>
          )}

          <FormField
            control={form.control}
            name='PayerScanEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable PayerScan')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Accept USDT and other crypto through PayerScan hosted checkout. Requires a merchant ID and an API key.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <SettingsFormGrid>
            <SettingsFormGridItem>
              <FormField
                control={form.control}
                name='PayerScanMerchantID'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('PayerScan merchant ID')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder='MID-XXXXXXXXXX'
                        autoComplete='off'
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Shown on the Store page in PayerScan. It must match the API key below.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGridItem>

            <SettingsFormGridItem>
              <FormField
                control={form.control}
                name='PayerScanApiKey'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('PayerScan API key')}</FormLabel>
                    <FormControl>
                      <Input
                        type='password'
                        placeholder={
                          props.hasStoredApiKey
                            ? t('Enter new key to update')
                            : t('Paste the store API key')
                        }
                        autoComplete='new-password'
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Sent as the x-api-key header and used to authenticate callbacks. Leave blank to keep the existing key.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGridItem>

            <SettingsFormGridItem>
              <FormField
                control={form.control}
                name='PayerScanUnitPrice'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Unit price (USD)')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        step={0.01}
                        min={0.01}
                        value={field.value}
                        onChange={(event) =>
                          field.onChange(
                            event.target.value === ''
                              ? 0
                              : event.target.valueAsNumber
                          )
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'USD charged per top-up unit before the group ratio and preset discount are applied.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGridItem>

            <SettingsFormGridItem>
              <FormField
                control={form.control}
                name='PayerScanMinTopUp'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum top-up quantity')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        step={1}
                        value={field.value}
                        onChange={(event) =>
                          field.onChange(
                            event.target.value === ''
                              ? 1
                              : event.target.valueAsNumber
                          )
                        }
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGridItem>
          </SettingsFormGrid>

          <FormField
            control={form.control}
            name='PayerScanBaseURL'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('PayerScan API endpoint')}</FormLabel>
                <FormControl>
                  <Input
                    type='url'
                    inputMode='url'
                    placeholder='https://api.payerscan.com'
                    autoComplete='off'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Leave blank to use the production endpoint https://api.payerscan.com.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormItem>
            <FormLabel>{t('Callback notification URL')}</FormLabel>
            <div className='flex items-center gap-2'>
              <Input readOnly value={props.webhookUrl} />
              <CopyButton
                value={props.webhookUrl}
                tooltip={t('Copy callback notification URL')}
              />
            </div>
            <FormDescription>
              {t(
                'Register this HTTPS endpoint as the callback URL of your PayerScan store.'
              )}
            </FormDescription>
          </FormItem>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
