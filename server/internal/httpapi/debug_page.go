package httpapi

import (
	"net/http"
)

func (s *Server) handleDebugPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(debugPageHTML))
}

const debugPageHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>circle-link local debug</title>
  <style>
    :root { color-scheme: light; }
    body {
      margin: 0;
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: linear-gradient(160deg, #f2efe8, #dfeae6);
      color: #16332d;
    }
    .wrap {
      max-width: 1120px;
      margin: 0 auto;
      padding: 32px 20px 48px;
    }
    h1, h2 { margin: 0 0 12px; }
    p { margin: 0 0 12px; }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 16px;
    }
    .card {
      background: rgba(255,255,255,0.78);
      backdrop-filter: blur(10px);
      border-radius: 18px;
      padding: 18px;
      box-shadow: 0 12px 30px rgba(22, 51, 45, 0.08);
    }
    label { display: block; font-size: 12px; margin-bottom: 6px; opacity: 0.8; }
    input, textarea, button {
      width: 100%;
      box-sizing: border-box;
      font: inherit;
      border-radius: 12px;
    }
    input, textarea {
      padding: 12px 14px;
      border: 1px solid #b9cec8;
      background: white;
      margin-bottom: 12px;
    }
    textarea { min-height: 100px; resize: vertical; }
    button {
      border: 0;
      background: #1e5b4f;
      color: white;
      padding: 12px 14px;
      cursor: pointer;
      margin-bottom: 10px;
    }
    button.secondary { background: #6d7d79; }
    .status {
      min-height: 24px;
      font-size: 14px;
      margin-bottom: 10px;
    }
    .mono {
      font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
      font-size: 12px;
      white-space: pre-wrap;
      word-break: break-word;
    }
    ul {
      list-style: none;
      padding: 0;
      margin: 0;
      display: grid;
      gap: 10px;
    }
    li {
      background: rgba(30, 91, 79, 0.07);
      border-radius: 14px;
      padding: 12px;
    }
    li.clickable { cursor: pointer; }
    .hint {
      font-size: 12px;
      opacity: 0.75;
      margin-top: 8px;
    }
  </style>
</head>
<body>
  <div class="wrap">
    <h1>circle-link local debug</h1>
    <p>Open this page in two browser tabs, create two users, then send messages between them.</p>
    <div class="grid">
      <section class="card">
        <h2>Account</h2>
        <div id="authStatus" class="status"></div>
        <label>Email</label>
        <input id="email" placeholder="alice@example.com" />
        <label>Password</label>
        <input id="password" type="password" placeholder="strong-password" />
        <label>Display name</label>
        <input id="displayName" placeholder="Alice" />
        <button id="signupBtn">Sign up</button>
        <label>Verification token</label>
        <input id="verificationToken" placeholder="token from signup response" />
        <button id="verifyBtn" class="secondary">Verify email</button>
        <button id="loginBtn">Login</button>
        <button id="meBtn" class="secondary">Load current user</button>
        <div id="meBox" class="mono"></div>
      </section>
      <section class="card">
        <h2>Device</h2>
        <div id="deviceStatus" class="status"></div>
        <button id="registerDeviceBtn">Register current device</button>
        <button id="loadDevicesBtn" class="secondary">Refresh devices</button>
        <div id="devicesBox" class="mono"></div>
      </section>
      <section class="card">
        <h2>Send message</h2>
        <div id="sendStatus" class="status"></div>
        <label>Recipient email</label>
        <input id="recipientEmail" placeholder="bob@example.com" />
        <label>Message</label>
        <textarea id="messageBody" placeholder="hello from circle-link"></textarea>
        <button id="sendBtn">Send</button>
      </section>
      <section class="card">
        <h2>Inbox</h2>
        <div id="inboxStatus" class="status"></div>
        <button id="refreshInboxBtn" class="secondary">Refresh inbox now</button>
        <ul id="inboxList"></ul>
        <p class="hint">Click an inbox message to emit a read receipt.</p>
      </section>
      <section class="card">
        <h2>Sent</h2>
        <div id="sentStatus" class="status"></div>
        <ul id="sentList"></ul>
      </section>
    </div>
  </div>
  <script>
    let accessToken = "";
    let currentUserId = "";
    let currentDeviceId = "";
    let messageSeq = 0;
    let socket = null;
    const inboxItems = new Map();
    const sentItems = new Map();

    const authStatus = document.getElementById("authStatus");
    const deviceStatus = document.getElementById("deviceStatus");
    const sendStatus = document.getElementById("sendStatus");
    const inboxStatus = document.getElementById("inboxStatus");
    const sentStatus = document.getElementById("sentStatus");
    const meBox = document.getElementById("meBox");
    const devicesBox = document.getElementById("devicesBox");
    const inboxList = document.getElementById("inboxList");
    const sentList = document.getElementById("sentList");

    function setStatus(node, message, isError = false) {
      node.textContent = message || "";
      node.style.color = isError ? "#a72525" : "#1e5b4f";
    }

    async function request(path, method = "GET", body = null) {
      const response = await fetch(path, {
        method,
        headers: {
          "Content-Type": "application/json",
          ...(accessToken ? { "Authorization": "Bearer " + accessToken } : {})
        },
        body: body ? JSON.stringify(body) : undefined
      });
      const payload = await response.json();
      if (payload.error) {
        throw new Error(payload.error.message || "Request failed");
      }
      return payload.data;
    }

    async function loadInbox() {
      if (!accessToken) {
        inboxList.innerHTML = "";
        return;
      }
      try {
        const path = currentDeviceId ? "/v1/messages?deviceId=" + encodeURIComponent(currentDeviceId) : "/v1/messages";
        const data = await request(path);
        inboxList.innerHTML = "";
        inboxItems.clear();
        (data.items || []).forEach(item => {
          upsertInboxItem(item, "snapshot", "append");
        });
        setStatus(inboxStatus, "Inbox refreshed");
      } catch (error) {
        setStatus(inboxStatus, error.message, true);
      }
    }

    function upsertInboxItem(item, label, mode = "prepend") {
      const merged = { ...(inboxItems.get(item.messageId) || {}), ...item };
      inboxItems.set(merged.messageId, merged);

      let li = Array.from(inboxList.children).find((node) => node.dataset.messageId === merged.messageId);
      if (!li) {
        li = document.createElement("li");
        li.dataset.messageId = merged.messageId;
        li.classList.add("clickable");
        if (mode === "append") {
          inboxList.appendChild(li);
        } else {
          inboxList.prepend(li);
        }
      }
      li.innerHTML = "<strong>" + merged.senderEmail + "</strong><br/>" +
        previewMessage(merged) + "<br/><span class='mono'>" + merged.sentAt + " · " + formatStatusLine(merged, label) + "</span>";
      li.onclick = () => markInboxMessageRead(merged.messageId);
    }

    function upsertSentItem(item, label = "") {
      const merged = { ...(sentItems.get(item.messageId) || {}), ...item };
      sentItems.set(merged.messageId, merged);

      let li = Array.from(sentList.children).find((node) => node.dataset.messageId === merged.messageId);
      if (!li) {
        li = document.createElement("li");
        li.dataset.messageId = merged.messageId;
        sentList.prepend(li);
      }

      const destination = merged.recipientEmail || merged.recipientUserId || "unknown recipient";
      li.innerHTML = "<strong>To " + destination + "</strong><br/>" +
        previewMessage(merged) + "<br/><span class='mono'>" + (merged.sentAt || new Date().toISOString()) + " · " +
        formatStatusLine(merged, label) + "</span>";
    }

    function formatStatusLine(item, label = "") {
      const parts = [];
      parts.push(item.deliveryStatus || "unknown");
      if (item.recipientDeviceId) {
        parts.push(item.recipientDeviceId);
      }
      if (label) {
        parts.push(label);
      }
      return parts.join(" · ");
    }

    function markInboxMessageRead(messageId) {
      const item = inboxItems.get(messageId);
      if (!item || item.deliveryStatus === "read") {
        return;
      }
      if (!socket || socket.readyState !== WebSocket.OPEN) {
        setStatus(inboxStatus, "WebSocket is required to send read receipts", true);
        return;
      }

      socket.send(JSON.stringify({
        type: "message.ack",
        payload: {
          messageId,
          status: "read"
        }
      }));
      upsertInboxItem({
        ...item,
        deliveryStatus: "read",
        readAt: new Date().toISOString()
      }, "read");
      setStatus(inboxStatus, "Read receipt sent for " + messageId);
    }

    function utf8ToBase64(value) {
      const bytes = new TextEncoder().encode(value);
      let binary = "";
      bytes.forEach((byte) => {
        binary += String.fromCharCode(byte);
      });
      return btoa(binary);
    }

    function base64ToUtf8(value) {
      const binary = atob(value);
      const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));
      return new TextDecoder().decode(bytes);
    }

    function previewMessage(item) {
      if (item.body) {
        return item.body;
      }
      if (item.header && item.header.encoding === "debug-base64-utf8" && item.ciphertext) {
        try {
          return base64ToUtf8(item.ciphertext);
        } catch (error) {
          return "[ciphertext decode failed]";
        }
      }
      return "[encrypted payload]";
    }

    function buildDebugEnvelope(plaintext) {
      return {
        header: {
          scheme: "debug-placeholder",
          encoding: "debug-base64-utf8",
          version: 1
        },
        ratchetPublicKey: "debug-rpk-" + (currentDeviceId || "web") + "-" + Date.now(),
        ciphertext: utf8ToBase64(plaintext)
      };
    }

    function connectWebSocket() {
      if (!accessToken || !currentUserId || !currentDeviceId) {
        return;
      }
      if (socket) {
        socket.close();
      }

      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      socket = new WebSocket(protocol + "//" + window.location.host + "/v1/ws");
      socket.onopen = () => {
        socket.send(JSON.stringify({
          type: "session.bind",
          payload: {
            accessToken,
            userId: currentUserId,
            deviceId: currentDeviceId
          }
        }));
        setStatus(inboxStatus, "WebSocket opened, binding session...");
      };
      socket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        switch (message.type) {
          case "session.bound":
            setStatus(inboxStatus, "Live WebSocket connected for " + message.payload.deviceId);
            break;
          case "message.mailbox":
            upsertInboxItem(message.payload, "mailbox", "append");
            break;
          case "message.deliver":
            upsertInboxItem(message.payload, "live");
            setStatus(inboxStatus, "New message received in real time");
            socket.send(JSON.stringify({
              type: "message.ack",
              payload: {
                messageId: message.payload.messageId,
                status: "delivered"
              }
            }));
            upsertInboxItem({
              ...message.payload,
              deliveryStatus: "delivered",
              deliveredAt: new Date().toISOString()
            }, "delivered");
            break;
          case "delivery.ack":
            setStatus(sendStatus, "Ack " + message.payload.status + " for " + message.payload.messageId);
            setStatus(sentStatus, "Latest sender update: " + message.payload.status + " for " + message.payload.messageId);
            upsertSentItem({
              messageId: message.payload.messageId,
              conversationId: message.payload.conversationId,
              senderUserId: message.payload.senderUserId,
              senderDeviceId: message.payload.senderDeviceId,
              recipientUserId: message.payload.recipientUserId,
              recipientDeviceId: message.payload.recipientDeviceId,
              clientMessageSeq: message.payload.clientMessageSeq,
              deliveryStatus: message.payload.status,
              deliveredAt: message.payload.status === "delivered" || message.payload.status === "read" ? message.payload.ackedAt : null,
              readAt: message.payload.status === "read" ? message.payload.ackedAt : null
            }, "ack");
            break;
          case "system.error":
            setStatus(inboxStatus, (message.payload && message.payload.message) || "WebSocket error", true);
            break;
        }
      };
      socket.onerror = () => {
        setStatus(inboxStatus, "WebSocket connection failed", true);
      };
      socket.onclose = () => {
        setStatus(inboxStatus, "WebSocket disconnected", true);
      };
    }

    document.getElementById("signupBtn").addEventListener("click", async () => {
      try {
        const data = await request("/v1/auth/signup", "POST", {
          email: document.getElementById("email").value,
          password: document.getElementById("password").value,
          displayName: document.getElementById("displayName").value
        });
        document.getElementById("verificationToken").value = data.verificationToken || "";
        setStatus(authStatus, "Signed up. Verification token filled for local testing.");
      } catch (error) {
        setStatus(authStatus, error.message, true);
      }
    });

    document.getElementById("verifyBtn").addEventListener("click", async () => {
      try {
        await request("/v1/auth/verify-email", "POST", {
          email: document.getElementById("email").value,
          verificationToken: document.getElementById("verificationToken").value
        });
        setStatus(authStatus, "Email verified.");
      } catch (error) {
        setStatus(authStatus, error.message, true);
      }
    });

    document.getElementById("loginBtn").addEventListener("click", async () => {
      try {
        const data = await request("/v1/auth/login", "POST", {
          email: document.getElementById("email").value,
          password: document.getElementById("password").value
        });
        accessToken = data.accessToken;
        currentUserId = data.userId;
        setStatus(authStatus, "Logged in.");
        await loadInbox();
        connectWebSocket();
      } catch (error) {
        setStatus(authStatus, error.message, true);
      }
    });

    document.getElementById("meBtn").addEventListener("click", async () => {
      try {
        const data = await request("/v1/me");
        meBox.textContent = JSON.stringify(data, null, 2);
      } catch (error) {
        setStatus(authStatus, error.message, true);
      }
    });

    document.getElementById("registerDeviceBtn").addEventListener("click", async () => {
      try {
        const now = Date.now();
        const data = await request("/v1/devices", "POST", {
          deviceName: "debug-browser",
          platform: "macos",
          pushToken: "",
          keyBundle: {
            identityKeyPublic: "identity-" + now,
            signedPrekeyPublic: "signed-" + now,
            signedPrekeySignature: "sig-" + now,
            signedPrekeyVersion: 1,
            oneTimePrekeys: ["otp-" + now + "-1", "otp-" + now + "-2"]
          }
        });
        currentDeviceId = data.deviceId;
        setStatus(deviceStatus, "Device registered: " + data.deviceId);
        connectWebSocket();
      } catch (error) {
        setStatus(deviceStatus, error.message, true);
      }
    });

    document.getElementById("loadDevicesBtn").addEventListener("click", async () => {
      try {
        const data = await request("/v1/devices");
        devicesBox.textContent = JSON.stringify(data.items || [], null, 2);
        if (!currentDeviceId && data.items && data.items.length > 0) {
          currentDeviceId = data.items[0].deviceId;
          connectWebSocket();
        }
      } catch (error) {
        setStatus(deviceStatus, error.message, true);
      }
    });

    document.getElementById("sendBtn").addEventListener("click", async () => {
      try {
        messageSeq += 1;
        const messageId = "msg_" + Date.now() + "_" + messageSeq;
        const recipientEmail = document.getElementById("recipientEmail").value;
        const plaintext = document.getElementById("messageBody").value;
        const envelope = buildDebugEnvelope(document.getElementById("messageBody").value);
        const payload = {
          messageId,
          conversationId: "conv_" + recipientEmail,
          recipientEmail,
          recipientDeviceId: "",
          contentType: "text/plain",
          clientMessageSeq: messageSeq,
          header: envelope.header,
          ratchetPublicKey: envelope.ratchetPublicKey,
          ciphertext: envelope.ciphertext
        };
        upsertSentItem({
          ...payload,
          body: plaintext,
          sentAt: new Date().toISOString(),
          deliveryStatus: "pending"
        }, "queued");
        if (socket && socket.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({
            type: "message.send",
            payload
          }));
          setStatus(sentStatus, "Queued " + messageId + " for relay");
        } else {
          const data = await request("/v1/messages", "POST", payload);
          setStatus(sendStatus, "Message sent: " + data.messageId);
          setStatus(sentStatus, "HTTP fallback stored " + data.messageId);
          upsertSentItem({ ...data, recipientEmail, body: plaintext }, "http");
        }
        document.getElementById("messageBody").value = "";
      } catch (error) {
        setStatus(sendStatus, error.message, true);
      }
    });

    document.getElementById("refreshInboxBtn").addEventListener("click", loadInbox);
  </script>
</body>
</html>`
