import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Get session ID from environment variable
const SESSION_ID = __ENV.SESSION_ID;
if (!SESSION_ID) {
    throw new Error('SESSION_ID environment variable must be set');
}

// Custom metrics for error tracking
const errorRate = new Rate('error_rate');

// Configuration for stepped load test
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
    
    // Connection settings
    batch: 20,  // Max number of simultaneous connections per VU

    scenarios: {
        stepped_load: {
            executor: 'ramping-arrival-rate',
            startRate: 100,
            timeUnit: '1s',
            preAllocatedVUs: 100,
            maxVUs: 1000000,
            stages: [
                // { duration: '30s', target: 100 },    // Start with 100 RPS
                // { duration: '30s', target: 500 },    // 500 RPS
                // { duration: '30s', target: 1000 },   // 1K RPS
                // { duration: '30s', target: 2000 },   // 2K RPS
                // { duration: '30s', target: 5000 },   // 5K RPS
                // { duration: '30s', target: 7500 },   // 7.5K RPS
                { duration: '30s', target: 10000 },  // 10K RPS
            ],
        },
    },
    thresholds: {
        'error_rate': [{
            threshold: 'rate <= 0.1',
            abortOnFail: true,
            delayAbortEval: '10s'
        }],
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
        
        // Check if response is not 200
        const isError = response.status !== 200;
        errorRate.add(isError);

        // Log errors for debugging
        if (isError && response.body) {
            console.log(
                `Error: Status ${response.status}, ` +
                `Body: ${response.body.length > 0 ? response.body.substring(0, 100) : 'empty'}`
            );
        }
    } catch (error) {
        console.log(`Request failed: ${error.message}`);
        errorRate.add(true);
    }
}

// Setup function to log test parameters
export function setup() {
    console.log('Starting stepped load test');
    console.log(`Session ID: ${SESSION_ID}`);
    return { startTime: new Date().toISOString() };
}

// Teardown function to log test completion
export function teardown(data) {
    console.log('Test completed');
    console.log(`Start time: ${data.startTime}`);
    console.log(`End time: ${new Date().toISOString()}`);
}
