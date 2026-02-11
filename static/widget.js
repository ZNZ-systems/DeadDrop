(function () {
  "use strict";

  // ---------------------------------------------------------------------------
  // 1. Locate our own <script> tag and extract configuration
  // ---------------------------------------------------------------------------
  var currentScript =
    document.currentScript ||
    (function () {
      var scripts = document.getElementsByTagName("script");
      return scripts[scripts.length - 1];
    })();

  var domainId = currentScript.getAttribute("data-deaddrop-id");
  if (!domainId) {
    console.error("[DeadDrop] Missing data-deaddrop-id attribute on script tag.");
    return;
  }

  // Derive the API host from the script's src attribute.
  var scriptSrc = currentScript.getAttribute("src");
  var apiBase = "";
  try {
    var url = new URL(scriptSrc, window.location.href);
    apiBase = url.origin;
  } catch (_) {
    // Fallback: strip the pathname portion manually.
    var a = document.createElement("a");
    a.href = scriptSrc;
    apiBase = a.protocol + "//" + a.host;
  }

  var ENDPOINT = apiBase + "/api/v1/messages";

  // ---------------------------------------------------------------------------
  // 2. Create the host element and Shadow DOM
  // ---------------------------------------------------------------------------
  var host = document.createElement("div");
  host.id = "deaddrop-widget-host";
  document.body.appendChild(host);

  var shadow = host.attachShadow({ mode: "closed" });

  // ---------------------------------------------------------------------------
  // 3. Styles (all inline inside the shadow root)
  // ---------------------------------------------------------------------------
  var style = document.createElement("style");
  style.textContent = [
    "*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }",

    /* Floating trigger button */
    ".dd-trigger {",
    "  position: fixed;",
    "  bottom: 24px;",
    "  right: 24px;",
    "  z-index: 2147483647;",
    "  width: 56px;",
    "  height: 56px;",
    "  border-radius: 50%;",
    "  border: none;",
    "  background: #1a1a2e;",
    "  color: #fff;",
    "  cursor: pointer;",
    "  box-shadow: 0 4px 14px rgba(0,0,0,0.25);",
    "  display: flex;",
    "  align-items: center;",
    "  justify-content: center;",
    "  transition: transform 0.2s ease, box-shadow 0.2s ease;",
    "}",
    ".dd-trigger:hover {",
    "  transform: scale(1.08);",
    "  box-shadow: 0 6px 20px rgba(0,0,0,0.35);",
    "}",
    ".dd-trigger svg {",
    "  width: 26px;",
    "  height: 26px;",
    "  fill: #fff;",
    "}",

    /* Panel overlay */
    ".dd-panel {",
    "  position: fixed;",
    "  bottom: 92px;",
    "  right: 24px;",
    "  z-index: 2147483646;",
    "  width: 370px;",
    "  max-width: calc(100vw - 48px);",
    "  max-height: calc(100vh - 120px);",
    "  border-radius: 12px;",
    "  overflow: hidden;",
    "  box-shadow: 0 8px 30px rgba(0,0,0,0.18);",
    "  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;",
    "  font-size: 14px;",
    "  line-height: 1.5;",
    "  opacity: 0;",
    "  transform: translateY(20px);",
    "  pointer-events: none;",
    "  transition: opacity 0.25s ease, transform 0.25s ease;",
    "}",
    ".dd-panel.dd-open {",
    "  opacity: 1;",
    "  transform: translateY(0);",
    "  pointer-events: auto;",
    "}",

    /* Header */
    ".dd-header {",
    "  background: #1a1a2e;",
    "  color: #fff;",
    "  padding: 18px 20px;",
    "  display: flex;",
    "  align-items: center;",
    "  justify-content: space-between;",
    "}",
    ".dd-header-title {",
    "  font-size: 16px;",
    "  font-weight: 600;",
    "}",
    ".dd-close {",
    "  background: none;",
    "  border: none;",
    "  color: #fff;",
    "  cursor: pointer;",
    "  padding: 4px;",
    "  line-height: 1;",
    "  font-size: 20px;",
    "  opacity: 0.7;",
    "  transition: opacity 0.15s;",
    "}",
    ".dd-close:hover { opacity: 1; }",

    /* Body */
    ".dd-body {",
    "  background: #fff;",
    "  padding: 20px;",
    "  overflow-y: auto;",
    "  max-height: calc(100vh - 220px);",
    "}",

    /* Form elements */
    ".dd-field {",
    "  margin-bottom: 14px;",
    "}",
    ".dd-label {",
    "  display: block;",
    "  font-size: 13px;",
    "  font-weight: 500;",
    "  color: #374151;",
    "  margin-bottom: 4px;",
    "}",
    ".dd-label .dd-optional {",
    "  font-weight: 400;",
    "  color: #9ca3af;",
    "  font-size: 12px;",
    "}",
    ".dd-input, .dd-textarea {",
    "  width: 100%;",
    "  padding: 10px 12px;",
    "  border: 1px solid #d1d5db;",
    "  border-radius: 8px;",
    "  font-size: 14px;",
    "  font-family: inherit;",
    "  color: #111827;",
    "  background: #f9fafb;",
    "  transition: border-color 0.15s, box-shadow 0.15s;",
    "  outline: none;",
    "}",
    ".dd-input:focus, .dd-textarea:focus {",
    "  border-color: #6366f1;",
    "  box-shadow: 0 0 0 3px rgba(99,102,241,0.15);",
    "  background: #fff;",
    "}",
    ".dd-textarea {",
    "  resize: vertical;",
    "  min-height: 100px;",
    "}",

    /* Honeypot */
    ".dd-hp {",
    "  position: absolute;",
    "  left: -9999px;",
    "  top: -9999px;",
    "  opacity: 0;",
    "  height: 0;",
    "  width: 0;",
    "  overflow: hidden;",
    "}",

    /* Submit button */
    ".dd-submit {",
    "  width: 100%;",
    "  padding: 12px;",
    "  background: #1a1a2e;",
    "  color: #fff;",
    "  border: none;",
    "  border-radius: 8px;",
    "  font-size: 15px;",
    "  font-weight: 600;",
    "  font-family: inherit;",
    "  cursor: pointer;",
    "  transition: background 0.15s;",
    "}",
    ".dd-submit:hover {",
    "  background: #16213e;",
    "}",
    ".dd-submit:disabled {",
    "  opacity: 0.6;",
    "  cursor: not-allowed;",
    "}",

    /* Feedback messages */
    ".dd-feedback {",
    "  padding: 12px 14px;",
    "  border-radius: 8px;",
    "  font-size: 13px;",
    "  margin-bottom: 14px;",
    "  display: none;",
    "}",
    ".dd-feedback.dd-success {",
    "  display: block;",
    "  background: #ecfdf5;",
    "  color: #065f46;",
    "  border: 1px solid #a7f3d0;",
    "}",
    ".dd-feedback.dd-error {",
    "  display: block;",
    "  background: #fef2f2;",
    "  color: #991b1b;",
    "  border: 1px solid #fecaca;",
    "}"
  ].join("\n");

  shadow.appendChild(style);

  // ---------------------------------------------------------------------------
  // 4. Build the DOM
  // ---------------------------------------------------------------------------

  // -- Helper: build the mail SVG icon safely (no innerHTML) --
  function createMailIcon() {
    var NS = "http://www.w3.org/2000/svg";
    var svg = document.createElementNS(NS, "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("aria-hidden", "true");
    var path = document.createElementNS(NS, "path");
    path.setAttribute(
      "d",
      "M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 " +
      "2-2V6c0-1.1-.9-2-2-2zm0 4l-8 5-8-5V6l8 5 8-5v2z"
    );
    svg.appendChild(path);
    return svg;
  }

  // Trigger button
  var trigger = document.createElement("button");
  trigger.className = "dd-trigger";
  trigger.setAttribute("aria-label", "Open contact form");
  trigger.appendChild(createMailIcon());
  shadow.appendChild(trigger);

  // Panel
  var panel = document.createElement("div");
  panel.className = "dd-panel";
  panel.setAttribute("role", "dialog");
  panel.setAttribute("aria-label", "Contact form");

  // Header
  var header = document.createElement("div");
  header.className = "dd-header";

  var headerTitle = document.createElement("span");
  headerTitle.className = "dd-header-title";
  headerTitle.textContent = "Send us a message";

  var closeBtn = document.createElement("button");
  closeBtn.className = "dd-close";
  closeBtn.setAttribute("aria-label", "Close contact form");
  closeBtn.textContent = "\u2715"; // Unicode multiplication sign (X)

  header.appendChild(headerTitle);
  header.appendChild(closeBtn);
  panel.appendChild(header);

  // Body
  var body = document.createElement("div");
  body.className = "dd-body";

  // Feedback area
  var feedback = document.createElement("div");
  feedback.className = "dd-feedback";
  body.appendChild(feedback);

  // Form
  var form = document.createElement("form");
  form.setAttribute("novalidate", "");

  // Hidden domain_id
  var domainInput = document.createElement("input");
  domainInput.type = "hidden";
  domainInput.name = "domain_id";
  domainInput.value = domainId;
  form.appendChild(domainInput);

  // Name field
  form.appendChild(buildField("name", "Name", "text", "Your name", true));

  // Email field
  form.appendChild(buildField("email", "Email", "email", "you@example.com", true));

  // Message field
  var msgField = document.createElement("div");
  msgField.className = "dd-field";
  var msgLabel = document.createElement("label");
  msgLabel.className = "dd-label";
  msgLabel.textContent = "Message";
  msgLabel.setAttribute("for", "dd-message");
  var msgTextarea = document.createElement("textarea");
  msgTextarea.className = "dd-textarea";
  msgTextarea.id = "dd-message";
  msgTextarea.name = "message";
  msgTextarea.placeholder = "How can we help?";
  msgTextarea.required = true;
  msgField.appendChild(msgLabel);
  msgField.appendChild(msgTextarea);
  form.appendChild(msgField);

  // Honeypot field (hidden from view and screen readers)
  var hpField = document.createElement("div");
  hpField.className = "dd-hp";
  hpField.setAttribute("aria-hidden", "true");
  var hpLabel = document.createElement("label");
  hpLabel.setAttribute("for", "dd-gotcha");
  hpLabel.textContent = "Leave this empty";
  var hpInput = document.createElement("input");
  hpInput.type = "text";
  hpInput.id = "dd-gotcha";
  hpInput.name = "_gotcha";
  hpInput.tabIndex = -1;
  hpInput.autocomplete = "off";
  hpField.appendChild(hpLabel);
  hpField.appendChild(hpInput);
  form.appendChild(hpField);

  // Submit button
  var submitBtn = document.createElement("button");
  submitBtn.type = "submit";
  submitBtn.className = "dd-submit";
  submitBtn.textContent = "Send Message";
  form.appendChild(submitBtn);

  body.appendChild(form);
  panel.appendChild(body);
  shadow.appendChild(panel);

  // ---------------------------------------------------------------------------
  // 5. Event handling
  // ---------------------------------------------------------------------------

  var isOpen = false;

  function togglePanel() {
    isOpen = !isOpen;
    panel.classList.toggle("dd-open", isOpen);
    trigger.setAttribute("aria-expanded", String(isOpen));
  }

  trigger.addEventListener("click", togglePanel);
  closeBtn.addEventListener("click", function () {
    if (isOpen) togglePanel();
  });

  form.addEventListener("submit", function (e) {
    e.preventDefault();

    // Client-side validation: message is required.
    if (!msgTextarea.value.trim()) {
      showFeedback("Please enter a message.", true);
      msgTextarea.focus();
      return;
    }

    submitBtn.disabled = true;
    submitBtn.textContent = "Sending\u2026";
    hideFeedback();

    // Build URL-encoded body.
    var params = new URLSearchParams();
    params.append("domain_id", domainId);
    params.append("name", form.elements["name"].value);
    params.append("email", form.elements["email"].value);
    params.append("message", msgTextarea.value);
    params.append("_gotcha", hpInput.value);

    var xhr = new XMLHttpRequest();
    xhr.open("POST", ENDPOINT, true);
    xhr.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");

    xhr.onload = function () {
      var res;
      try {
        res = JSON.parse(xhr.responseText);
      } catch (_) {
        res = {};
      }

      if (xhr.status >= 200 && xhr.status < 300 && res.ok) {
        showFeedback("Message sent! Thank you.", false);
        form.reset();
        domainInput.value = domainId; // restore hidden field after reset
      } else {
        var msg = (res && res.error) ? res.error : "Something went wrong. Please try again.";
        showFeedback(msg, true);
      }
      submitBtn.disabled = false;
      submitBtn.textContent = "Send Message";
    };

    xhr.onerror = function () {
      showFeedback("Network error. Please check your connection and try again.", true);
      submitBtn.disabled = false;
      submitBtn.textContent = "Send Message";
    };

    xhr.send(params.toString());
  });

  // ---------------------------------------------------------------------------
  // 6. Helpers
  // ---------------------------------------------------------------------------

  function buildField(name, labelText, type, placeholder, optional) {
    var field = document.createElement("div");
    field.className = "dd-field";

    var label = document.createElement("label");
    label.className = "dd-label";
    label.setAttribute("for", "dd-" + name);
    label.textContent = labelText + " ";
    if (optional) {
      var optSpan = document.createElement("span");
      optSpan.className = "dd-optional";
      optSpan.textContent = "(optional)";
      label.appendChild(optSpan);
    }

    var input = document.createElement("input");
    input.className = "dd-input";
    input.type = type;
    input.id = "dd-" + name;
    input.name = name;
    input.placeholder = placeholder || "";

    field.appendChild(label);
    field.appendChild(input);
    return field;
  }

  function showFeedback(message, isError) {
    feedback.textContent = message;
    feedback.className = "dd-feedback " + (isError ? "dd-error" : "dd-success");
  }

  function hideFeedback() {
    feedback.className = "dd-feedback";
    feedback.textContent = "";
  }
})();
