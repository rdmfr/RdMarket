import { z } from 'zod';
import { createEnvelopeSchema } from '@/schemas/market';
import { apiClient, validateResponse } from './api';
import { EconomicIndicatorSchema } from '@/schemas/economic';

const EconomicIndicatorsEnvelope = createEnvelopeSchema(z.array(EconomicIndicatorSchema));

export const economicApi = {
  async getIndicators() {
    const raw = await apiClient.get('economic/indicators').json();
    const parsed = validateResponse(EconomicIndicatorsEnvelope, raw);
    if (parsed.error) throw new Error(parsed.error.message);
    return parsed.data;
  },
};
