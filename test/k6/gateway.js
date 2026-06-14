import http from 'k6/http';
import { check, sleep } from 'k6';

const baseUrl = __ENV.VEXOR_URL || 'http://127.0.0.1:8080';
const routePath = __ENV.VEXOR_PATH || '/users';

const vus = Number(__ENV.VUS || 50);
const durationSeconds = Number(__ENV.DURATION_SECONDS || 30);

export const options = {
  scenarios: {
    gateway_load: {
      executor: 'constant-vus',
      vus,
      duration: `${durationSeconds}s`,
    },
  },

  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

export default function () {
  const url = `${baseUrl}${routePath}`;

  const res = http.get(url, {
    tags: {
      route: routePath,
    },
  });

  const success = check(res, {
    'status is 2xx or 3xx': (r) =>
      r.status >= 200 && r.status < 400,
  });

  if (!success && __VU === 1) {
    console.log(
      `Request failed | status=${res.status} | body=${String(
        res.body
      ).slice(0, 200)}`
    );
  }

  sleep(1);
}