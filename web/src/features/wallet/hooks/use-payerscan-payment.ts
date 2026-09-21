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
import i18next from 'i18next'
import { useState, useCallback } from 'react'
import { toast } from 'sonner'

import { isHttpUrl } from '@/lib/content-format'
import { handleServerError } from '@/lib/handle-server-error'

import { requestPayerScanPayment, isApiSuccess } from '../api'

function getCheckoutUrl(data: unknown): string | null {
  if (!data || typeof data !== 'object') {
    return null
  }

  if ('checkout_url' in data && typeof data.checkout_url === 'string') {
    return data.checkout_url.trim()
  }

  return null
}

function getErrorMessage(message: string | undefined, data: unknown): string {
  if (typeof data === 'string' && data.trim()) {
    return data
  }

  return (
    (message && message !== 'success' ? message : undefined) ||
    i18next.t('Payment request failed')
  )
}

/**
 * Hook for the PayerScan hosted crypto checkout flow.
 *
 * Same-tab redirect (window.location.href) rather than window.open: the
 * user-gesture context is lost across the await, so popups get blocked.
 */
export function usePayerScanPayment() {
  const [processing, setProcessing] = useState(false)

  const processPayerScanPayment = useCallback(async (topupAmount: number) => {
    setProcessing(true)

    try {
      const response = await requestPayerScanPayment({
        amount: Math.floor(topupAmount),
      })

      if (isApiSuccess(response)) {
        const checkoutUrl = getCheckoutUrl(response.data)

        if (checkoutUrl) {
          // Backend-provided redirect target: reject anything that is not
          // http(s) so a malformed value cannot become a javascript: URL.
          if (!isHttpUrl(checkoutUrl)) {
            toast.error(i18next.t('Invalid payment redirect URL'))
            return false
          }
          toast.success(i18next.t('Redirecting to payment page...'))
          window.location.href = checkoutUrl
          return true
        }
      }

      handleServerError(response, undefined, {
        title: getErrorMessage(response.message, response.data),
      })
      return false
    } catch (error) {
      handleServerError(error, i18next.t('Payment request failed'))
      return false
    } finally {
      setProcessing(false)
    }
  }, [])

  return { processing, processPayerScanPayment }
}
