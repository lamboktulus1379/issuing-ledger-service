import http from 'k6/http';
import { check, sleep } from 'k6';
import exec from 'k6/execution';

export const options = {
    vus: 50,          // 50 concurrent virtual users
    duration: '30s',  // Run the test for 30 seconds
    thresholds: {
        // Enforce the sub-50ms enterprise requirement
        http_req_duration: ['p(99)<50'], 
    },
};

export default function () {
    const url = 'http://localhost:10001/api/authorize';
    
    // Create a fast, guaranteed unique string (e.g., "5-102" for VU 5, Iteration 102)
    const uniqueSuffix = `${exec.vu.idInInstance}-${exec.scenario.iterationInInstance}`;
    
   // Create a fast, guaranteed unique string (e.g., "5-102" for VU 5, Iteration 102)
    const randomSource = `acc_${Math.floor(Math.random() * 1000) + 1}`;
    const randomDest = `acc_${Math.floor(Math.random() * 1000) + 1}`;

    // Ensure they aren't the same account
    if (randomSource === randomDest) return; 

    const payload = JSON.stringify({
        transaction_id: `txn-cdc-${uniqueSuffix}`,
        source_account_id: randomSource,
        destination_account_id: randomDest,
        amount: 120,
        currency: 'USD'
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3OTAzNTU0OTMsImlzcyI6IjIiLCJ1c2VyX25hbWUiOiJqb2huZG9lIn0.cipxCF2qOiw8PKVZWKthoTaclIwJiSDIgdghC47LtrA',
            'Idempotency-Key': `req-uuid-${uniqueSuffix}`
        },
    };

    const res = http.post(url, payload, params);

    check(res, {
        'is status 200': (r) => r.status === 200,
    });

    // A small sleep prevents the script from overloading your local network stack
    sleep(0.05); 
}