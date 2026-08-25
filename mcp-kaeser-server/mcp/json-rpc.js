"use strict";

function hasRequestId(request) {
    return Object.prototype.hasOwnProperty.call(request, "id");
}

function createResponder(writeResponse) {
    function sendResult(request, result) {
        if (!hasRequestId(request)) return;
        writeResponse({ jsonrpc: "2.0", id: request.id, result });
    }

    function sendRequestError(request, code, message, data) {
        if (!hasRequestId(request)) return;
        const error = { code, message };
        if (data !== undefined) error.data = data;
        writeResponse({ jsonrpc: "2.0", id: request.id, error });
    }

    function sendProtocolError(id, code, message, data) {
        const error = { code, message };
        if (data !== undefined) error.data = data;
        writeResponse({ jsonrpc: "2.0", id, error });
    }

    return { sendProtocolError, sendRequestError, sendResult };
}

const stdioResponder = createResponder((response) => {
    process.stdout.write(`${JSON.stringify(response)}\n`);
});

module.exports = {
    createResponder,
    hasRequestId,
    stdioResponder
};
