const numerator = require("./numerator");
const express = require("express");
const api = express();
const port = 3000;
const morgan = require('morgan')

api.use(express.json());

morgan.token('body', (req) => {
    // Only log body for methods that usually have one
    if (req.method === 'POST' || req.method === 'PUT' || req.method === 'PATCH') {
        return JSON.stringify(req.body);
    }
    return 'N/A'; // For GET and other methods, return N/A
});

morgan.token('req-content-type', (req) => req.headers['content-type'] || 'N/A');
morgan.token('res-content-type', (req, res) => res.getHeader('content-type') || 'N/A');
api.use(
    morgan(
        '=====\n' +
        ':method :url :status\n' +
        ':response-time ms\n' +
        'Response Size: :res[content-length] bytes\n' +
        'Request Body: :body\n' +
        'Request Size: :req[content-length] bytes\n' +
        'Request Content-Type: :req-content-type\n' +
        'Response Content-Type: :res-content-type\n' +
        '====='
    )
);

// Get the current numerator
api.get("/numerator", async (req, res) => {
    try {
        const currentNumerator = await numerator.getNumerator();
        res.json({numerator: currentNumerator});
    } catch (error) {
        res.status(500).json({error: "Failed to retrieve numerator."});
    }
});

// Set the numerator
api.put("/numerator", async (req, res) => {
    const {value} = req.body;
    if (typeof value !== "number" || isNaN(value)) {
        return res.status(400).json({error: "Invalid value for numerator."});
    }
    try {
        await numerator.setNumerator(value);
        res.json({numerator: value});
    } catch (error) {
        res.status(500).json({error: "Failed to set numerator."});
    }
});

// Test and set the numerator atomically
api.put("/numerator/test-and-set", async (req, res) => {
    const {oldValue, newValue} = req.body;
    if (typeof oldValue !== "number" || isNaN(oldValue) || typeof newValue !== "number" || isNaN(newValue)) {
        return res.status(400).json({error: "Invalid values for test-and-set."});
    }
    try {
        const result = await numerator.testAndSetNumerator(newValue, oldValue);
        if (result === -1) {
            return res.status(400).json({
                error: "Numerator does not match the expected old value.",
                currentNumerator: await numerator.getNumerator(),
            });
        }
        res.json({numerator: result});
    } catch (error) {
        res.status(500).json({error: "Failed to perform test-and-set operation."});
    }
});

// Lock the numerator
api.post("/numerator/lock", async (req, res) => {
    let timeout = 10000; // Default value
    if (req.body && req.body.timeout !== undefined) {
        timeout = req.body.timeout;
    }

    try {
        await numerator.lockNumerator(timeout);
        res.json({message: "Lock acquired"});
    } catch (error) {
        res.status(408).json({error: error.message, stack: error.stack});
    }
});

// Unlock the numerator
api.delete("/numerator/lock", (req, res) => {
    try {
        numerator.unlockNumerator();
        res.json({message: "Lock released"});
    } catch (error) {
        res.status(500).json({error: "Failed to release lock."});
    }
});

api.listen(port, () => {
    console.log(`numerator-api is running on http://localhost:${port}`);
});
