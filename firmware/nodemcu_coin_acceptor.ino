/*
 * NodeMCU (ESP8266) Coin Acceptor Firmware
 * 
 * Reads coin pulses from a physical coin acceptor via GPIO interrupt,
 * looks up the coin value, and sends an HTTP POST to the SBC server
 * to add time to the active session.
 *
 * Hardware:
 *   - NodeMCU ESP8266 (e.g. Amica, LoLin)
 *   - Coin acceptor with pulse output (e.g. CH-926)
 *   - Pulse wire -> D2 (GPIO4)
 *   - GND wire -> GND
 *
 * Configuration (edit below before flashing):
 *   - WiFi SSID and password
 *   - SBC server IP and port
 *   - Device ID (unique per device)
 *   - API key (if device is registered with one)
 *   - Session ID (set via serial command or hardcoded)
 */

#include <ESP8266WiFi.h>
#include <ESP8266HTTPClient.h>
#include <ArduinoJson.h>

// ========== Configuration ==========
const char* WIFI_SSID     = "YOUR_WIFI_SSID";
const char* WIFI_PASSWORD  = "YOUR_WIFI_PASSWORD";
const char* SERVER_URL     = "http://192.168.1.100:8080";  // SBC server
const char* DEVICE_ID      = "nodemcu-001";
const char* API_KEY        = "";  // Leave empty if no API key required

// Coin acceptor settings
const int COIN_PIN    = 4;        // D2 = GPIO4
const int DEBOUNCE_MS = 50;       // Debounce time in milliseconds
const int PULSES_PER_COIN = 1;    // Number of pulses per coin (adjust per your acceptor)

// ========== Globals ==========
volatile int pulseCount = 0;
int lastCoinValue = 0;
unsigned long lastPulseTime = 0;
int currentSessionID = 0;
bool wifiConnected = false;

// ========== Interrupt Handler ==========
ICACHE_RAM_ATTR void coinPulseISR() {
  unsigned long now = millis();
  if (now - lastPulseTime > DEBOUNCE_MS) {
    pulseCount++;
    lastPulseTime = now;
  }
}

// ========== WiFi Connection ==========
void connectWiFi() {
  Serial.print("Connecting to WiFi: ");
  Serial.println(WIFI_SSID);
  
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  
  int attempts = 0;
  while (WiFi.status() != WL_CONNECTED && attempts < 30) {
    delay(500);
    Serial.print(".");
    attempts++;
  }
  
  if (WiFi.status() == WL_CONNECTED) {
    Serial.println("\nWiFi connected!");
    Serial.print("IP: ");
    Serial.println(WiFi.localIP());
    wifiConnected = true;
  } else {
    Serial.println("\nWiFi connection failed. Retrying in 5s...");
    wifiConnected = false;
    delay(5000);
  }
}

// ========== Send Coin Event to Server ==========
bool sendCoinEvent(int coinValue, int sessionID) {
  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("WiFi not connected, cannot send coin event");
    return false;
  }
  
  if (sessionID <= 0) {
    Serial.println("No active session, coin rejected");
    return false;
  }
  
  WiFiClient client;
  HTTPClient http;
  
  String url = String(SERVER_URL) + "/api/coin-event";
  http.begin(client, url);
  http.addHeader("Content-Type", "application/json");
  
  if (API_KEY[0] != '\0') {
    http.addHeader("X-API-Key", API_KEY);
  }
  
  // Build JSON payload
  StaticJsonDocument<200> doc;
  doc["session_id"] = sessionID;
  doc["coin_value"] = coinValue;
  doc["device_id"]  = DEVICE_ID;
  
  String payload;
  serializeJson(doc, payload);
  
  Serial.print("Sending coin event: ");
  Serial.println(payload);
  
  int httpCode = http.POST(payload);
  
  if (httpCode == HTTP_CODE_OK) {
    String response = http.getString();
    Serial.println("Coin accepted! Server response: " + response);
    http.end();
    return true;
  } else {
    Serial.print("HTTP error: ");
    Serial.println(httpCode);
    if (httpCode > 0) {
      Serial.println("Response: " + http.getString());
    }
    http.end();
    return false;
  }
}

// ========== Process Coin Pulses ==========
void processCoins() {
  if (pulseCount >= PULSES_PER_COIN) {
    int coinValue = pulseCount / PULSES_PER_COIN;
    pulseCount = pulseCount % PULSES_PER_COIN;  // Keep remainder
    
    Serial.print("Coin detected! Value: ");
    Serial.print(coinValue);
    Serial.print(" peso(s), Session: ");
    Serial.println(currentSessionID);
    
    // Blink LED to confirm coin accepted
    digitalWrite(LED_BUILTIN, HIGH);
    delay(100);
    digitalWrite(LED_BUILTIN, LOW);
    
    if (currentSessionID > 0) {
      bool success = sendCoinEvent(coinValue, currentSessionID);
      if (!success) {
        Serial.println("WARNING: Failed to send coin event to server!");
        // Could store locally and retry here if needed
      }
    } else {
      Serial.println("No active session. Coin pulse ignored.");
    }
  }
}

// ========== Serial Commands ==========
// Send "S<session_id>" via serial to set the active session
// Example: "S42" sets session ID to 42
void checkSerialCommands() {
  if (Serial.available() > 0) {
    String cmd = Serial.readStringUntil('\n');
    cmd.trim();
    
    if (cmd.startsWith("S")) {
      currentSessionID = cmd.substring(1).toInt();
      Serial.print("Session ID set to: ");
      Serial.println(currentSessionID);
    } else if (cmd == "status") {
      Serial.print("Session: ");
      Serial.println(currentSessionID);
      Serial.print("WiFi: ");
      Serial.println(wifiConnected ? "Connected" : "Disconnected");
      Serial.print("Pulse count: ");
      Serial.println(pulseCount);
    } else if (cmd == "help") {
      Serial.println("Commands:");
      Serial.println("  S<id>  - Set session ID (e.g. S42)");
      Serial.println("  status - Show current status");
      Serial.println("  help   - Show this help");
    }
  }
}

// ========== Setup ==========
void setup() {
  Serial.begin(115200);
  delay(100);
  
  Serial.println("\n=============================");
  Serial.println("PisoWiFi Coin Acceptor");
  Serial.println("Device: " + String(DEVICE_ID));
  Serial.println("=============================");
  
  // Configure coin pulse pin
  pinMode(COIN_PIN, INPUT_PULLUP);
  attachInterrupt(digitalPinToInterrupt(COIN_PIN), coinPulseISR, FALLING);
  
  // Built-in LED for feedback
  pinMode(LED_BUILTIN, OUTPUT);
  
  // Connect to WiFi
  connectWiFi();
  
  Serial.println("\nReady! Send 'help' via serial for commands.");
  Serial.println("Set session with: S<session_id>");
}

// ========== Loop ==========
void loop() {
  // Check WiFi connection
  if (WiFi.status() != WL_CONNECTED) {
    wifiConnected = false;
    connectWiFi();
  }
  
  // Process any coin pulses
  processCoins();
  
  // Check for serial commands
  checkSerialCommands();
  
  // Small delay to prevent watchdog issues
  delay(10);
}
