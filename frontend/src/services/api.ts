import ky from 'ky';

export const apiClient = ky.create({
  prefix: '/api/v1',
  timeout: 10000,
  retry: {
    limit: 2,
    methods: ['get'],
    statusCodes: [408, 500, 502, 503, 504],
  },
  headers: {
    'Accept': 'application/json',
  },
});
