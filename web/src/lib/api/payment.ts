import { api } from './client'

export const paymentApi = {
  remitaInit: (amount: number, orderId: string) =>
    api.post<{ rrr: string; amount: number }>('/payment/remita/init', {
      amount,
      order_id: orderId,
    }),

  remitaVerify: (rrr: string) =>
    api.post<{ paid: boolean }>('/payment/remita/verify', { rrr }),
}
