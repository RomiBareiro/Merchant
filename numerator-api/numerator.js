let numerator = 3;
let lock = false;

/*
 Explanation of the module. This is a mock service to obtain unique numerators. All functions should be considered
 async even if they are not in this example. In a real use case, this numerator would be maintained in an external source
 (e.g., a REDIS server), and all operations would involve network access.

 The testAndSetNumerator function operates atomically; if the current numerator value in the remote repository
 (in this example, REDIS) matches the parameter value, it sets the numerator to the new value. If it changes, it returns -1
 (the numerator cannot have negative values).

 The lockNumerator function ensures a single entry point but only for the lockNumerator function. If one request calls
 lockNumerator and another request calls lockNumerator afterward, the second will wait until the first calls unlockNumerator
 or until a timeout occurs. It's important to note that calling lockNumerator does not affect the rest of the functions.
*/

function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}

async function getNumerator() {
    return numerator;
}

async function setNumerator(value) {
    numerator = value;
}

async function testAndSetNumerator(newValue, oldValue) {
    if (numerator === oldValue) {
        numerator = newValue;
        return numerator;
    }
    return -1;
}

async function lockNumerator(timeout) {
    const lapse = 400;
    const maxRetries = Math.floor(timeout / lapse) + 1;
    console.log(`Trying to lock with timeout ${timeout} - retries ${maxRetries}`);
    for (let attempts = 0; attempts < maxRetries; attempts++) {
        console.log(`Attempt ${attempts + 1} of ${maxRetries}`);

        if (!lock) {
            console.log("Lock acquired");
            lock = true;
            return;
        }

        await sleep(lapse);
    }
    console.log("Error while locking");
    throw new Error("Timeout while locking");
}

async function unlockNumerator() {
    lock = false;
}

module.exports = {
    getNumerator,
    setNumerator,
    testAndSetNumerator,
    lockNumerator,
    unlockNumerator
};
