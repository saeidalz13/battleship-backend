# Simplifying Action Codes

Currently, there exist codes to track the success or failure of requests, along with their purpose (e.g., create, attack, etc.). This document aims to simplify these codes by proposing solutions that eliminate the need to track the state of the request and its type (request vs. response).

## Solution 1 

### Eliminating `req` and `resp`

Given that the client is indifferent to the type of `message`—whether it's `incoming` or `outgoing`, in other words, `req` or `resp`—there's no necessity for distinct codes like `CodeReqCreateGame`.

### Removing `success` and `failure`

The success or failure of a request can be encapsulated within the `message` itself. In the case of success, the `message` would contain an empty `error_payload`, whereas in failure, it would include the `error_payload`.

## Solution 2 

This approach establishes a clear separation between incoming and outgoing messages, leveraging the strong type system of Golang to send only messages of type `outgoing`, while keeping the distinction between `req` and `resp` as currently implemented.

### Distinguishing `incoming` and `outgoing`

In this application, messages fall into two categories:

1. **Incoming**: Messages sent solely by the client (e.g., `create`, `ready`). These messages consist only of `code` and `payload`.

2. **Outgoing**: Messages dispatched exclusively by the server (e.g., `RespCreate`, `SelectGrid`). These messages include `code`, `payload`, and, in the event of failure, `error_payload`.




