import http from 'k6/http';
import { check } from 'k6';
import { Rate } from 'k6/metrics';

// Get session ID from environment variable
const SESSION_ID = __ENV.SESSION_ID;
if (!SESSION_ID) {
    throw new Error('SESSION_ID environment variable must be set');
}

// Custom metrics for error tracking
const errorRate = new Rate('error_rate');

// Configuration for constant load test
export const options = {
    // DNS settings
    dns: {
        ttl: '1m',
        select: 'roundRobin',
        policy: 'preferIPv4',
    },

    // Network settings
    userAgent: 'K6LoadTest/1.0',
    noConnectionReuse: false,
    noVUConnectionReuse: false,
    
    // Timeout settings
    setupTimeout: '30s',
    teardownTimeout: '30s',

    scenarios: {
        constant_load: {
            executor: 'constant-arrival-rate',
            rate: 10000,           // 10K RPS
            timeUnit: '1s',        // per second
            duration: '120s',       // run for 2 minutes
            preAllocatedVUs: 50,  // initial VUs
            maxVUs: 1000,         // maximum VUs if needed
        },
    },

    thresholds: {
        'error_rate': [{
            threshold: 'rate <= 0.1',  // 10% error rate threshold
            abortOnFail: true,
            delayAbortEval: '10s'
        }],
        http_req_duration: ['p(95)<2000'], // 95% of requests should be under 2s
    },
};

export default function () {
    const url = 'http://127.0.0.1:8100/api/points/get?user_id=1';
    
    const params = {
        timeout: '5s',
        connecting_timeout: '3s',
    };
    
    const headers = {
        'user_id': '1',
        'Cookie': `session_id=${SESSION_ID}`,
        'Content-Type': 'application/json',
        'Accept': 'application/json',
    };

    try {
        const response = http.get(url, {
            headers,
            timeout: params.timeout,
            connecting_timeout: params.connecting_timeout
        });
        
        // Check response
        const checkResult = check(response, {
            'status is 200': (r) => r.status === 200,
            'response time < 2s': (r) => r.timings.duration < 2000,
            'has valid response': (r) => r.body !== null && r.body.length > 0,
        });

        errorRate.add(!checkResult);

        if (!checkResult) {
            if (response.body) {
                console.log(
                    `Error: Status ${response.status}, ` +
                    `Duration: ${response.timings.duration}ms, ` +
                    `Body: ${response.body.length > 0 ? response.body.substring(0, 100) : 'empty'}`
                );
            } else {
                console.log(
                    `Error: Status ${response.status}, ` +
                    `Duration: ${response.timings.duration}ms, ` +
                    `No response body`
                );
            }
        }
    } catch (error) {
        console.log(`Request failed: ${error.message}`);
        errorRate.add(true);
    }
}

// Setup function to log test parameters
export function setup() {
    console.log('Starting constant load test at 10K RPS');
    console.log(`Session ID: ${SESSION_ID}`);
    return { startTime: new Date().toISOString() };
}

// Teardown function to log test completion
export function teardown(data) {
    console.log('Test completed');
    console.log(`Start time: ${data.startTime}`);
    console.log(`End time: ${new Date().toISOString()}`);
}
