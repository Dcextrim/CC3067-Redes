"use strict";

const { hasRequestId } = require("./json-rpc.js");
const { createRequestHandler } = require("./request-handler.js");
const { isPlainObject, validateRequestEnvelope } = require("./validation.js");

function createMessageHandler(responder) {
    const handleRequest = createRequestHandler(responder);

    return function handleMessage(request) {
        const validationError = validateRequestEnvelope(request);
        if (validationError) {
            const id = isPlainObject(request) && hasRequestId(request) && ["string", "number"].includes(typeof request.id)
                ? request.id
                : null;
            responder.sendProtocolError(id, -32600, "Invalid Request", validationError);
            return;
        }

        handleRequest(request);
    };
}

module.exports = { createMessageHandler };
