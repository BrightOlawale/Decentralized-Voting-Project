#include <lvgl.h>
#include <TFT_eSPI.h>
#include <XPT2046_Touchscreen.h>
#include <Arduino.h>
#include <vector>
#include <Adafruit_Fingerprint.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <ArduinoJson.h>
#include <mbedtls/sha256.h>
#include <WiFiClientSecure.h>
#include <time.h>

// Touchscreen pins
#define XPT2046_IRQ 36
#define XPT2046_MOSI 32
#define XPT2046_MISO 39
#define XPT2046_CLK 25
#define XPT2046_CS 33

// Fingerprint sensor pins
#define RXD2 27
#define TXD2 22

#define SCREEN_WIDTH 240
#define SCREEN_HEIGHT 320

// Hardware setup
SPIClass touchscreenSPI = SPIClass(VSPI);
XPT2046_Touchscreen touchscreen(XPT2046_CS, XPT2046_IRQ);
HardwareSerial fpSerial(2);
Adafruit_Fingerprint finger = Adafruit_Fingerprint(&fpSerial);

#define DRAW_BUF_SIZE (SCREEN_WIDTH * SCREEN_HEIGHT / 10 * (LV_COLOR_DEPTH / 8))
uint32_t draw_buf[DRAW_BUF_SIZE / 4];

// Vote counting (local backup)
int votes_candidate_a = 0;
int votes_candidate_b = 0;

const char* WIFI_SSID = "COL";
const char* WIFI_PASSWORD = "Act4:12";

// API config
const char* API_BASE_URL = "https://blockchain-voting-system-v1-0-0.onrender.com";
const char* TOKEN_ENDPOINT = "/api/v1/public/token/terminal";
const char* ENSURE_PU_ENDPOINT = "/api/v1/terminal/polling-unit/ensure";
const char* REGISTER_ENDPOINT = "/api/v1/public/voter/register";
const char* VOTE_ENDPOINT = "/api/v1/voting/cast";

// Session
String g_jwt = "";
String g_device_id = "";
String g_polling_unit_id = "Pollin-Unit-OAU";
String g_config_election_id = ""; // configured via UI

// Candidate cache (RAM)
static std::vector<String> g_candidates;
static String g_current_election_id = "";
static unsigned long g_candidates_fetched_ms = 0;
static const unsigned long CANDIDATES_TTL_MS = 300000; // 5 min

// Storage for registered users (matric -> fingerprint ID mapping)
struct RegisteredUser {
    String matric_number;
    uint8_t fingerprint_id;
    String nin; // Added nin field
};

#define MAX_USERS 80
#define MAX_VOTED 160
static RegisteredUser registered_users[MAX_USERS];
static uint16_t registered_users_count = 0;
static uint8_t voted_fingerprint_ids[MAX_VOTED];
static uint16_t voted_count = 0;
static uint8_t next_fingerprint_id = 1;

// Current user data during registration
char current_matric_number[20] = {0}; // e.g. "CSC/2018/003"
uint8_t current_fingerprint_id = 0;

// Screen objects
lv_obj_t *home_screen = NULL;
lv_obj_t *register_screen = NULL;
lv_obj_t *personal_info_screen = NULL;
lv_obj_t *vote_screen = NULL;
lv_obj_t *fingerprint_screen = NULL;
lv_obj_t *wifi_screen = NULL;
lv_obj_t *config_screen = NULL;
lv_obj_t *candidates_container = NULL;
lv_obj_t *verify_btn = NULL;

// UI elements for registration
lv_obj_t *ta_matric;
lv_obj_t *ta_nin_reg;
lv_obj_t *kb;
lv_obj_t *label_result;
lv_obj_t *label_vote_result_a;
lv_obj_t *label_vote_result_b;
lv_obj_t *label_fingerprint_status;
lv_obj_t *ta_vote_nin; // input for NIN during voting
String current_vote_nin = "";
lv_obj_t *wifi_status_label_home = NULL;
lv_timer_t *wifi_refresh_timer = NULL;
String current_registration_nin = "";

// Context for WiFi dialog callbacks
struct WifiDialogCtx {
    lv_obj_t *mbox;
    char ssid[64];
    int channel;
    uint8_t bssid[6];
    bool hasBssid;
};

// Context for WiFi list button (per network)
struct WifiBtnCtx {
    char ssid[64];
    int channel;
    uint8_t bssid[6];
    bool hasBssid;
};

// Context for Config screen keyboard submission
struct ConfigDialogCtx {
    lv_obj_t *ta_eid;
    lv_obj_t *ta_pu;
};

// Config verification state
static bool g_election_verified = false;
static bool g_pu_verified = false;
static lv_obj_t *status_eid_label = NULL;
static lv_obj_t *status_pu_label = NULL;
static bool g_time_synced = false;

// Forward declarations
void go_to_homepage();
void create_register_screen();
void create_personal_info_screen();
void show_vote_screen();
void show_fingerprint_screen(bool is_registration);
static void refresh_wifi_label_cb(lv_timer_t * t);

// API Functions
bool sendRegistrationToAPI(const RegisteredUser& user) {
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("WiFi not connected, cannot send to API");
        return false;
    }
    
    HTTPClient http;
    String url = String(API_BASE_URL) + String(REGISTER_ENDPOINT);
    http.begin(url);
    http.addHeader("Content-Type", "application/json");
    if (g_jwt.length() > 0) {
        http.addHeader("Authorization", "Bearer " + g_jwt);
    }
    
    // Create JSON payload (use placeholders for missing personal fields)
    StaticJsonDocument<256> doc;
    String nin = user.matric_number;
    String fpHash = sha256Hex(nin + "|" + String(user.fingerprint_id));
    doc["nin"] = nin;
    doc["first_name"] = "Unknown";
    doc["last_name"] = "Unknown";
    doc["date_of_birth"] = "1990-01-01T00:00:00Z";
    doc["gender"] = "Other";
    doc["fingerprint_data"] = fpHash;
    doc["polling_unit_id"] = g_polling_unit_id;
    
    String jsonString;
    serializeJson(doc, jsonString);
    
    Serial.println("Sending registration to API:");
    Serial.println("URL: " + url);
    Serial.println("Payload: " + jsonString);
    
    int httpResponseCode = http.POST(jsonString);
    
    if (httpResponseCode > 0) {
        String response = http.getString();
        Serial.println("API Response Code: " + String(httpResponseCode));
        Serial.println("API Response: " + response);
        
        if (httpResponseCode == 200 || httpResponseCode == 201 || httpResponseCode == 202) {
            http.end();
            return true;
        }
    } else {
        Serial.println("Error in HTTP request: " + String(httpResponseCode));
    }
    
    http.end();
    return false;
}

bool sendVoteToAPI(uint8_t fingerprint_id, const String& matric_number, const String& candidate) {
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("WiFi not connected, cannot send vote to API");
        return false;
    }
    
    HTTPClient http;
    String url = String(API_BASE_URL) + String(VOTE_ENDPOINT);
    http.begin(url);
    http.addHeader("Content-Type", "application/json");
    if (g_jwt.length() > 0) {
        http.addHeader("Authorization", "Bearer " + g_jwt);
    }
    
    // Create JSON payload
    StaticJsonDocument<192> doc;
    String fpHash = sha256Hex(matric_number + "|" + String(fingerprint_id));
    doc["fingerprint_data"] = fpHash;
    doc["nin"] = matric_number;
    doc["candidate_id"] = candidate;
    doc["polling_unit_id"] = g_polling_unit_id;
    
    String jsonString;
    serializeJson(doc, jsonString);
    
    Serial.println("Sending vote to API:");
    Serial.println("URL: " + url);
    Serial.println("Payload: " + jsonString);
    
    int httpResponseCode = http.POST(jsonString);
    // If unauthorized, try to refresh token once and retry
    if (httpResponseCode == 401) {
        http.end();
        Serial.println("JWT expired/invalid. Refreshing token...");
        if (WiFi.status() == WL_CONNECTED) {
            extern bool fetchTerminalToken();
            if (fetchTerminalToken()) {
                HTTPClient http2;
                http2.begin(url);
                http2.addHeader("Content-Type", "application/json");
                if (g_jwt.length() > 0) http2.addHeader("Authorization", "Bearer " + g_jwt);
                httpResponseCode = http2.POST(jsonString);
                String _ = http2.getString();
                http2.end();
            }
        }
    }
    
    if (httpResponseCode > 0) {
        String response = http.getString();
        Serial.println("Vote API Response Code: " + String(httpResponseCode));
        Serial.println("Vote API Response: " + response);
        
        if (httpResponseCode == 200 || httpResponseCode == 201 || httpResponseCode == 202) {
            http.end();
            return true;
        }
    } else {
        Serial.println("Error in vote HTTP request: " + String(httpResponseCode));
    }
    
    http.end();
    return false;
}

// Helper functions for fingerprint management
bool has_voted(uint8_t fingerprint_id) {
    Serial.print("Checking if fingerprint ID ");
    Serial.print(fingerprint_id);
    Serial.print(" has voted: ");
    for (uint16_t i = 0; i < voted_count; i++) if (voted_fingerprint_ids[i] == fingerprint_id) return true;
    Serial.println("NO");
    return false;
}

bool has_registered_matric(const String& matric) {
    Serial.print("Checking if matric ");
    Serial.print(matric);
    Serial.print(" is registered: ");
    for (uint16_t i = 0; i < registered_users_count; i++) {
        if (registered_users[i].matric_number == matric) {
            Serial.println("YES");
            return true;
        }
    }
    Serial.println("NO");
    return false;
}

uint8_t get_fingerprint_id_for_matric(const String& matric) {
    Serial.print("Getting fingerprint ID for matric ");
    Serial.print(matric);
    Serial.print(": ");
    for (uint16_t i = 0; i < registered_users_count; i++) {
        if (registered_users[i].matric_number == matric) {
            Serial.println(registered_users[i].fingerprint_id);
            return registered_users[i].fingerprint_id;
        }
    }
    Serial.println("NOT FOUND");
    return 0;
}

// Logging
void log_print(lv_log_level_t level, const char *buf) {
    LV_UNUSED(level);
    Serial.println(buf);
    Serial.flush();
}

// WIFI 
void connectToWiFi() {
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
    Serial.print("Connecting to WiFi");
    
    int attempts = 0;
    while (WiFi.status() != WL_CONNECTED && attempts < 20) {
        delay(500);
        Serial.print(".");
        attempts++;
    }
    
    if (WiFi.status() == WL_CONNECTED) {
        Serial.println();
        Serial.println("WiFi connected!");
        Serial.print("IP address: ");
        Serial.println(WiFi.localIP());
        Serial.print("MAC address: ");
        Serial.println(WiFi.macAddress());
    } else {
        Serial.println();
        Serial.println("Failed to connect to WiFi. Running in offline mode.");
    }
}

static void refresh_wifi_label_cb(lv_timer_t * t) {
    (void)t;
    if (!wifi_status_label_home) return;
    if (WiFi.status() == WL_CONNECTED) {
        String ip = WiFi.localIP().toString();
        String ssid = WiFi.SSID();
        String text = String("WiFi: ") + ssid + "  " + ip;
        lv_label_set_text(wifi_status_label_home, text.c_str());
    } else {
        lv_label_set_text(wifi_status_label_home, "WiFi: disconnected");
    }
}

// ===== API helpers =====
static String macNoColons(const String& mac){ String out; out.reserve(mac.length()); for(size_t i=0;i<mac.length();++i){ if(mac[i] != ':') out += mac[i]; } return out; }
static String buildPollingUnitIdFromMac(const String& mac){ return String("PU-") + macNoColons(mac); }

static bool ensureTimeSyncedOnce() {
    if (g_time_synced) return true;
    Serial.println("Syncing time via NTP...");
    configTime(0, 0, "pool.ntp.org", "time.nist.gov");
    const unsigned long start = millis();
    const unsigned long timeoutMs = 8000;
    time_t now = 0; struct tm timeinfo = {};
    while (millis() - start < timeoutMs) {
        time(&now);
        localtime_r(&now, &timeinfo);
        if (timeinfo.tm_year >= (2020 - 1900)) {
            g_time_synced = true;
            break;
        }
        delay(200);
    }
    if (g_time_synced) {
        char buf[64]; strftime(buf, sizeof(buf), "%Y-%m-%d %H:%M:%S", &timeinfo);
        Serial.print("Time synced: "); Serial.println(buf);
    } else {
        Serial.println("Time sync timed out; proceeding anyway");
    }
    return g_time_synced;
}

// Minimal URL parser to extract scheme, host, port, and path
static bool parseUrl(const String& url, String& scheme, String& host, uint16_t& port, String& uri) {
    int pos = url.indexOf("://");
    if (pos < 0) return false;
    scheme = url.substring(0, pos);
    int start = pos + 3;
    int slash = url.indexOf('/', start);
    String authority = (slash >= 0) ? url.substring(start, slash) : url.substring(start);
    int colon = authority.indexOf(':');
    if (colon >= 0) {
        host = authority.substring(0, colon);
        port = (uint16_t)authority.substring(colon + 1).toInt();
    } else {
        host = authority;
        port = (scheme == "https") ? 443 : 80;
    }
    uri = (slash >= 0) ? url.substring(slash) : String("/");
    return host.length() > 0;
}

// Read a single CRLF-terminated line from client (without CRLF)
static String readLine(WiFiClient& client) {
    String line;
    while (client.connected()) {
        int c = client.read();
        if (c < 0) { delay(1); continue; }
        if (c == '\r') continue;
        if (c == '\n') break;
        line += (char)c;
    }
    return line;
}

// Very simple chunked de-encoder for HTTP/1.1 responses
static bool dechunkBody(const String& chunked, String& out) {
    out = "";
    size_t i = 0;
    while (i < chunked.length()) {
        // Read chunk size line
        size_t lineEnd = chunked.indexOf("\n", i);
        if (lineEnd == (size_t)-1) return false;
        String sizeLine = chunked.substring(i, lineEnd);
        sizeLine.trim();
        uint32_t size = (uint32_t) strtoul(sizeLine.c_str(), NULL, 16);
        i = lineEnd + 1;
        if (size == 0) return true; // done
        if (i + size > chunked.length()) return false;
        out += chunked.substring(i, i + size);
        i += size;
        // Skip CRLF after chunk
        if (i + 1 < chunked.length() && chunked[i] == '\r' && chunked[i+1] == '\n') i += 2;
        else if (i < chunked.length() && (chunked[i] == '\n' || chunked[i] == '\r')) i++;
    }
    return true;
}

static bool httpPostJson(const String& path, const String& json, String& respBody, int& code, bool withAuth) {
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("HTTP POST aborted: WiFi not connected");
        return false;
    }
    HTTPClient http;
    String url = String(API_BASE_URL) + path;
    Serial.println("==== HTTP POST ====");
    Serial.print("URL: "); Serial.println(url);
    Serial.print("Auth: "); Serial.println(withAuth && g_jwt.length() > 0 ? "Bearer present" : "none");
    Serial.print("Payload: "); Serial.println(json);
    bool okBegin = false;
    String scheme, host, uri;
    uint16_t port = 0;
    if (!parseUrl(url, scheme, host, port, uri)) {
        Serial.println("URL parse failed");
        return false;
    }
    Serial.print("Host: "); Serial.println(host);
    Serial.print("Port: "); Serial.println(port);
    Serial.print("URI: "); Serial.println(uri);
    if (scheme == "https") {
        ensureTimeSyncedOnce();
        WiFiClientSecure client;
        client.setInsecure();
        client.setTimeout(15000);
        if (!client.connect(host.c_str(), port)) {
            Serial.println("TLS connect failed");
            code = -1; respBody = ""; return false;
        }
        // Build request
        String req;
        req.reserve(128 + json.length());
        req += "POST "; req += uri; req += " HTTP/1.1\r\n";
        req += "Host: "; req += host; req += "\r\n";
        req += "User-Agent: ESP32\r\n";
        req += "Accept: application/json\r\n";
        req += "Content-Type: application/json\r\n";
        req += "Connection: close\r\n";
        if (withAuth && g_jwt.length() > 0) { req += "Authorization: Bearer "; req += g_jwt; req += "\r\n"; }
        req += "Content-Length: "; req += String(json.length()); req += "\r\n\r\n";
        req += json;
        client.print(req);
        // Parse status line
        String status = readLine(client);
        int sp1 = status.indexOf(' '); int sp2 = status.indexOf(' ', sp1+1);
        code = (sp1>0 && sp2>sp1) ? status.substring(sp1+1, sp2).toInt() : -1;
        // Read headers
        bool chunked = false; int contentLength = -1;
        while (client.connected()) {
            String h = readLine(client);
            if (h.length() == 0) break;
            String hl = h; hl.toLowerCase();
            if (hl.startsWith("transfer-encoding:") && hl.indexOf("chunked") >= 0) chunked = true;
            if (hl.startsWith("content-length:")) contentLength = h.substring(h.indexOf(':')+1).toInt();
        }
        // Read body
        String raw;
        if (contentLength >= 0) {
            raw.reserve(contentLength);
            while ((int)raw.length() < contentLength && client.connected()) {
                int c = client.read(); if (c < 0) { delay(1); continue; } raw += (char)c;
            }
        } else {
            while (client.connected()) { int c = client.read(); if (c < 0) { delay(1); continue; } raw += (char)c; }
        }
        client.stop();
        if (chunked) {
            String de; if (dechunkBody(raw, de)) respBody = de; else respBody = raw;
        } else {
            respBody = raw;
        }
        Serial.print("Status: "); Serial.println(code);
        Serial.print("Response: "); Serial.println(respBody);
        Serial.println("==== END POST ====");
        return code > 0;
    } else {
        // Plain HTTP via HTTPClient
        bool ok = http.begin(host.c_str(), port, uri);
        if (!ok) { Serial.println("HTTP begin() failed"); return false; }
        http.addHeader("Content-Type", "application/json");
        if (withAuth && g_jwt.length() > 0) http.addHeader("Authorization", String("Bearer ") + g_jwt);
        http.setTimeout(15000);
        code = http.POST(json);
        respBody = http.getString();
        if (code <= 0) { Serial.print("ErrorStr: "); Serial.println(http.errorToString(code)); }
        http.end();
        Serial.print("Status: "); Serial.println(code);
        Serial.print("Response: "); Serial.println(respBody);
        Serial.println("==== END POST ====");
        return code > 0;
    }
}

static bool httpGet(const String& path, String& respBody, int& code) {
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("HTTP GET aborted: WiFi not connected");
        return false;
    }
    HTTPClient http;
    String url = String(API_BASE_URL) + path;
    Serial.println("==== HTTP GET ====");
    Serial.print("URL: "); Serial.println(url);
    bool okBegin = false;
    String scheme, host, uri;
    uint16_t port = 0;
    if (!parseUrl(url, scheme, host, port, uri)) {
        Serial.println("URL parse failed");
        return false;
    }
    Serial.print("Host: "); Serial.println(host);
    Serial.print("Port: "); Serial.println(port);
    Serial.print("URI: "); Serial.println(uri);
    if (scheme == "https") {
        ensureTimeSyncedOnce();
        WiFiClientSecure client;
        client.setInsecure();
        client.setTimeout(15000);
        if (!client.connect(host.c_str(), port)) {
            Serial.println("TLS connect failed");
            code = -1; respBody = "";
            return false;
        }
        // Build GET request
        String req;
        req.reserve(128);
        req += "GET "; req += uri; req += " HTTP/1.1\r\n";
        req += "Host: "; req += host; req += "\r\n";
        req += "User-Agent: ESP32\r\n";
        req += "Accept: application/json\r\n";
        req += "Connection: close\r\n\r\n";
        client.print(req);

        // Parse status line
        String status = readLine(client);
        int sp1 = status.indexOf(' '); int sp2 = status.indexOf(' ', sp1+1);
        code = (sp1>0 && sp2>sp1) ? status.substring(sp1+1, sp2).toInt() : -1;
        // Read headers
        bool chunked = false; int contentLength = -1;
        while (client.connected()) {
            String h = readLine(client);
            if (h.length() == 0) break;
            String hl = h; hl.toLowerCase();
            if (hl.startsWith("transfer-encoding:") && hl.indexOf("chunked") >= 0) chunked = true;
            if (hl.startsWith("content-length:")) contentLength = h.substring(h.indexOf(':')+1).toInt();
        }
        // Read body
        String raw;
        if (contentLength >= 0) {
            raw.reserve(contentLength);
            while ((int)raw.length() < contentLength && client.connected()) {
                int c = client.read(); if (c < 0) { delay(1); continue; } raw += (char)c;
            }
        } else {
            while (client.connected()) { int c = client.read(); if (c < 0) { delay(1); continue; } raw += (char)c; }
        }
        client.stop();
        if (chunked) {
            String de; if (dechunkBody(raw, de)) respBody = de; else respBody = raw;
        } else {
            respBody = raw;
        }
        Serial.print("Status: "); Serial.println(code);
        Serial.print("Response: "); Serial.println(respBody);
        Serial.println("==== END GET ====");
        return code > 0;
    } else {
        // Plain HTTP via HTTPClient
        bool ok = http.begin(host.c_str(), port, uri);
        if (!ok) { Serial.println("HTTP begin() failed"); return false; }
        http.setTimeout(15000);
        code = http.GET();
        respBody = http.getString();
        if (code <= 0) { Serial.print("ErrorStr: "); Serial.println(http.errorToString(code)); }
        http.end();
        Serial.print("Status: "); Serial.println(code);
        Serial.print("Response: "); Serial.println(respBody);
        Serial.println("==== END GET ====");
        return code > 0;
    }
}

static String sha256Hex(const String& s){
    unsigned char out[32];
    mbedtls_sha256_context ctx;
    mbedtls_sha256_init(&ctx);
    mbedtls_sha256_starts(&ctx, 0);
    mbedtls_sha256_update(&ctx, (const unsigned char*)s.c_str(), s.length());
    mbedtls_sha256_finish(&ctx, out);
    mbedtls_sha256_free(&ctx);
    char buf[65];
    for (int i = 0; i < 32; ++i) sprintf(&buf[i*2], "%02x", out[i]);
    buf[64] = 0;
    return String(buf);
}

static String extractJsonField(const String& body, const char* field){
    String key = String("\"") + field + "\"";
    int pos = body.indexOf(key);
    if (pos < 0) return String("");
    pos = body.indexOf(':', pos); if (pos < 0) return String("");
    // skip spaces
    while (pos+1 < (int)body.length() && (body[pos+1] == ' ')) pos++;
    int q1 = body.indexOf('"', pos+1); if (q1 < 0) return String("");
    int q2 = body.indexOf('"', q1+1); if (q2 < 0) return String("");
    return body.substring(q1+1, q2);
}

// Verification helpers
static bool verifyElectionId(const String& electionId) {
    if (electionId.length() == 0) return false;
    String body; int code = 0;
    bool ok = httpGet(String("/api/v1/public/election/") + electionId, body, code);
    return ok && code == 200;
}

static bool fetchTerminalToken() {
    String payload = String("{") + "\"device_id\":\"" + g_device_id + "\"" + "}";
    String resp; int code = 0;
    if (!httpPostJson(TOKEN_ENDPOINT, payload, resp, code, /*withAuth*/false)) return false;
    if (code >= 200 && code < 300) {
        String token = extractJsonField(resp, "token");
        if (token.length() == 0) {
            // try nested data.token
            int d = resp.indexOf("\"data\"");
            if (d >= 0) {
                int tk = resp.indexOf("\"token\"", d);
                if (tk >= 0) {
                    int c = resp.indexOf(':', tk); int q1 = resp.indexOf('"', c+1); int q2 = resp.indexOf('"', q1+1);
                    if (q1>0 && q2>q1) token = resp.substring(q1+1, q2);
                }
            }
        }
        if (token.length() > 0) { g_jwt = token; return true; }
    }
    Serial.printf("Token error (%d): %s\n", code, resp.c_str());
    return false;
}

static bool ensurePollingUnit() {
    String json = String("{") +
        "\"id\":\"" + g_polling_unit_id + "\"," +
        "\"name\":\"Terminal " + g_device_id + "\"," +
        "\"location\":\"Unknown\"," +
        "\"total_voters\":1000" +
        "}";
    String resp; int code = 0;
    if (!httpPostJson(ENSURE_PU_ENDPOINT, json, resp, code, /*withAuth*/false)) return false;
    return code >= 200 && code < 300;
}

static String get_matric_for_fp(uint8_t fid){
    for (uint16_t i = 0; i < registered_users_count; i++) if (registered_users[i].fingerprint_id == fid) return registered_users[i].matric_number;
    return String("");
}

static bool fetchCandidatesByElectionId(const String& electionId) {
    if (electionId.length() == 0) return false;
    Serial.print("Fetching candidates for election: "); Serial.println(electionId);
    String body; int code = 0;
    if (!httpGet(String("/api/v1/public/election/") + electionId + "/candidates", body, code)) return false;
    Serial.print("Candidates GET status: "); Serial.println(code);
    if (code != 200) return false;
    StaticJsonDocument<1024> doc;
    DeserializationError err = deserializeJson(doc, body);
    if (err) { Serial.print("JSON parse error: "); Serial.println(err.c_str()); return false; }
    JsonVariant data = doc["data"];
    if (data.isNull()) return false;
    g_candidates.clear();
    JsonArray cands = data["candidates"].as<JsonArray>();
    if (cands.isNull()) { cands = data.as<JsonArray>(); }
    if (!cands.isNull()) {
        for (JsonVariant v : cands) {
            const char* s = v.as<const char*>();
            if (s && *s) g_candidates.push_back(String(s));
        }
    }
    g_current_election_id = electionId;
    g_candidates_fetched_ms = millis();
    Serial.print("Candidates fetched: "); Serial.println((int)g_candidates.size());
    return g_candidates.size() > 0;
}

// Fetch candidates for the current active election (no manual ID required)
static bool fetchCandidatesForCurrent() {
    Serial.println("Fetching candidates for current election");
    String body; int code = 0;
    if (!httpGet(String("/api/v1/public/election/current"), body, code)) return false;
    Serial.print("Current election GET status: "); Serial.println(code);
    if (code != 200) return false;
    StaticJsonDocument<768> doc;
    if (deserializeJson(doc, body)) return false;
    String eid = "";
    JsonVariant data = doc["data"];
    if (!data.isNull()) {
        const char* id1 = data["id"] | (const char*)nullptr;
        const char* id2 = data["election_id"] | (const char*)nullptr;
        if (id1 && *id1) eid = String(id1);
        else if (id2 && *id2) eid = String(id2);
    }
    // Fill candidates directly if present
    g_candidates.clear();
    if (!data.isNull()) {
        JsonArray cands = data["candidates"].as<JsonArray>();
        if (!cands.isNull()) {
            for (JsonVariant v : cands) {
                const char* s = v.as<const char*>();
                if (s && *s) g_candidates.push_back(String(s));
            }
        }
    }
    if (!g_candidates.empty()) {
        g_current_election_id = eid;
        g_candidates_fetched_ms = millis();
        Serial.print("Candidates from current endpoint: "); Serial.println((int)g_candidates.size());
        return true;
    }
    if (eid.length() == 0) return false;
    g_config_election_id = eid;
    Serial.print("Resolved current election id: "); Serial.println(eid);
    return fetchCandidatesByElectionId(eid);
}

// Touchscreen input handling
void touchscreen_read(lv_indev_t *indev, lv_indev_data_t *data) {
    if (touchscreen.tirqTouched() && touchscreen.touched()) {
        TS_Point p = touchscreen.getPoint();
        int x = map(p.x, 200, 3700, 1, SCREEN_WIDTH);
        int y = map(p.y, 240, 3800, 1, SCREEN_HEIGHT);

        data->state = LV_INDEV_STATE_PRESSED;
        data->point.x = x;
        data->point.y = y;
    } else {
        data->state = LV_INDEV_STATE_RELEASED;
    }
}

// Enhanced validation functions
bool is_valid_matric(const String& matric) {
    Serial.print("Validating matric number: ");
    Serial.println(matric);
    
    if (matric.length() != 12) {
        Serial.println("Validation FAILED: Length is not 12");
        return false;
    }
    
    for (int i = 0; i < 3; ++i) {
        if (!isUpperCase(matric[i]) || !isAlpha(matric[i])) {
            Serial.print("Validation FAILED: Character at position ");
            Serial.print(i);
            Serial.println(" is not uppercase letter");
            return false;
        }
    }
    
    if (matric[3] != '/') {
        Serial.println("Validation FAILED: Character at position 3 is not '/' ");
        return false;
    }
    
    for (int i = 4; i < 8; ++i) {
        if (!isDigit(matric[i])) {
            Serial.print("Validation FAILED: Character at position ");
            Serial.print(i);
            Serial.println(" is not a digit");
            return false;
        }
    }
    
    if (matric[8] != '/') {
        Serial.println("Validation FAILED: Character at position 8 is not '/' ");
        return false;
    }
    
    for (int i = 9; i < 12; ++i) {
        if (!isDigit(matric[i])) {
            Serial.print("Validation FAILED: Character at position ");
            Serial.print(i);
            Serial.println(" is not a digit");
            return false;
        }
    }
    
    Serial.println("Matric validation PASSED");
    return true;
}

bool is_valid_name(const String& name) {
    if (name.length() < 2 || name.length() > 50) {
        return false;
    }
    for (int i = 0; i < name.length(); i++) {
        if (!isAlpha(name[i]) && name[i] != ' ' && name[i] != '-') {
            return false;
        }
    }
    return true;
}

bool is_valid_date(const String& date) {
    // Basic validation for YYYY-MM-DD format
    if (date.length() != 10) return false;
    if (date[4] != '-' || date[7] != '-') return false;
    
    // Check if year, month, day are numbers
    for (int i = 0; i < 4; i++) if (!isDigit(date[i])) return false;
    for (int i = 5; i < 7; i++) if (!isDigit(date[i])) return false;
    for (int i = 8; i < 10; i++) if (!isDigit(date[i])) return false;
    
    int year = date.substring(0, 4).toInt();
    int month = date.substring(5, 7).toInt();
    int day = date.substring(8, 10).toInt();
    
    if (year < 1950 || year > 2010) return false;  // Reasonable age range
    if (month < 1 || month > 12) return false;
    if (day < 1 || day > 31) return false;
    
    return true;
}

// Fingerprint functions (keeping your existing implementation)
uint8_t enroll_fingerprint(uint8_t id) {
    Serial.println("=== STARTING FINGERPRINT ENROLLMENT ===");
    Serial.print("Enrolling fingerprint with ID: ");
    Serial.println(id);
    
    int p = -1;
    Serial.print("Waiting for valid finger to enroll as #"); 
    Serial.println(id);
    
    lv_label_set_text(label_fingerprint_status, "Place finger on sensor...");
    lv_task_handler();
    
    while (p != FINGERPRINT_OK) {
        p = finger.getImage();
        lv_task_handler();
        delay(50);
        
        if (p == FINGERPRINT_OK) {
            Serial.println("First fingerprint image captured successfully");
            lv_label_set_text(label_fingerprint_status, "Image taken! Processing...");
            lv_task_handler();
            break;
        } else if (p == FINGERPRINT_NOFINGER) {
            // Silent wait for finger
        } else {
            Serial.print("Error taking first image, code: ");
            Serial.println(p);
            lv_label_set_text(label_fingerprint_status, "Error taking image, try again");
            lv_task_handler();
            delay(2000);
            return p;
        }
    }

    // Convert first image
    Serial.println("Converting first image to template...");
    p = finger.image2Tz(1);
    if (p != FINGERPRINT_OK) {
        Serial.print("Error converting first image, code: ");
        Serial.println(p);
        lv_label_set_text(label_fingerprint_status, "Error converting image");
        return p;
    }
    Serial.println("First image converted successfully");

    lv_label_set_text(label_fingerprint_status, "Remove finger...");
    lv_task_handler();
    Serial.println("Waiting for finger to be removed...");
    delay(2000);
    
    p = 0;
    while (p != FINGERPRINT_NOFINGER) {
        p = finger.getImage();
        lv_task_handler();
        delay(50);
    }
    Serial.println("Finger removed, ready for second capture");

    lv_label_set_text(label_fingerprint_status, "Place same finger again...");
    lv_task_handler();
    Serial.println("Waiting for second fingerprint capture...");
    
    p = -1;
    while (p != FINGERPRINT_OK) {
        p = finger.getImage();
        lv_task_handler();
        delay(50);
        
        if (p == FINGERPRINT_OK) {
            Serial.println("Second fingerprint image captured successfully");
            lv_label_set_text(label_fingerprint_status, "Second image taken!");
            lv_task_handler();
            break;
        } else if (p == FINGERPRINT_NOFINGER) {
            // Silent wait for finger
        } else {
            Serial.print("Error taking second image, code: ");
            Serial.println(p);
            lv_label_set_text(label_fingerprint_status, "Error taking second image");
            return p;
        }
    }

    // Convert second image
    Serial.println("Converting second image to template...");
    p = finger.image2Tz(2);
    if (p != FINGERPRINT_OK) {
        Serial.print("Error converting second image, code: ");
        Serial.println(p);
        lv_label_set_text(label_fingerprint_status, "Error converting second image");
        return p;
    }
    Serial.println("Second image converted successfully");

    // Create model
    lv_label_set_text(label_fingerprint_status, "Creating fingerprint model...");
    lv_task_handler();
    Serial.println("Creating fingerprint model from two templates...");
    
    p = finger.createModel();
    if (p == FINGERPRINT_OK) {
        Serial.println("Fingerprint model created successfully - prints matched!");
        lv_label_set_text(label_fingerprint_status, "Fingerprints matched!");
    } else if (p == FINGERPRINT_ENROLLMISMATCH) {
        Serial.println("ERROR: Fingerprints did not match during enrollment");
        lv_label_set_text(label_fingerprint_status, "Fingerprints did not match, try again");
        return p;
    } else {
        Serial.print("Error creating fingerprint model, code: ");
        Serial.println(p);
        lv_label_set_text(label_fingerprint_status, "Error creating model");
        return p;
    }

    // Store model
    Serial.print("Storing fingerprint model with ID: ");
    Serial.println(id);
    p = finger.storeModel(id);
    if (p == FINGERPRINT_OK) {
        Serial.println("Fingerprint stored successfully in sensor database");
        lv_label_set_text(label_fingerprint_status, "Fingerprint stored successfully!");
        lv_task_handler();
        Serial.println("=== FINGERPRINT ENROLLMENT COMPLETED ===");
        return FINGERPRINT_OK;
    } else {
        Serial.print("Error storing fingerprint, code: ");
        Serial.println(p);
        lv_label_set_text(label_fingerprint_status, "Error storing fingerprint");
        return p;
    }
}

int verify_fingerprint() {
    Serial.println("=== STARTING FINGERPRINT VERIFICATION ===");
    
    lv_label_set_text(label_fingerprint_status, "Place finger on sensor for verification...");
    lv_task_handler();
    
    uint8_t p = finger.getImage();
    int timeout_count = 0;
    
    Serial.println("Waiting for finger placement...");
    while (p == FINGERPRINT_NOFINGER && timeout_count < 100) {
        p = finger.getImage();
        lv_task_handler();
        delay(100);
        timeout_count++;
    }
    
    if (p != FINGERPRINT_OK) {
        Serial.print("Error reading fingerprint for verification, code: ");
        Serial.println(p);
        lv_label_set_text(label_fingerprint_status, "Error reading fingerprint");
        return -1;
    }

    Serial.println("Fingerprint image captured for verification");
    lv_label_set_text(label_fingerprint_status, "Image taken, processing...");
    lv_task_handler();

    Serial.println("Converting image to template...");
    p = finger.image2Tz();
    if (p != FINGERPRINT_OK) {
        Serial.print("Error converting verification image, code: ");
        Serial.println(p);
        lv_label_set_text(label_fingerprint_status, "Error converting image");
        return -1;
    }
    Serial.println("Image converted to template successfully");

    Serial.println("Searching for fingerprint match in database...");
    p = finger.fingerFastSearch();
    if (p == FINGERPRINT_OK) {
        Serial.print("MATCH FOUND! Fingerprint ID: ");
        Serial.print(finger.fingerID);
        Serial.print(" with confidence: ");
        Serial.println(finger.confidence);
        lv_label_set_text_fmt(label_fingerprint_status, "Match found! ID: %d", finger.fingerID);
        lv_task_handler();
        Serial.println("=== FINGERPRINT VERIFICATION SUCCESSFUL ===");
        return finger.fingerID;
    } else if (p == FINGERPRINT_NOTFOUND) {
        Serial.println("NO MATCH FOUND - fingerprint not in database");
        lv_label_set_text(label_fingerprint_status, "No match found");
        return -1;
    } else {
        Serial.print("Search error during verification, code: ");
        Serial.println(p);
        lv_label_set_text(label_fingerprint_status, "Search error");
        return -1;
    }
}

// Vote counting update
void update_vote_results() {
    Serial.println("Updating vote results display...");
    Serial.print("Candidate A votes: ");
    Serial.println(votes_candidate_a);
    Serial.print("Candidate B votes: ");
    Serial.println(votes_candidate_b);
    
    if (label_vote_result_a) lv_label_set_text_fmt(label_vote_result_a, "Candidate A: %d votes", votes_candidate_a);
    if (label_vote_result_b) lv_label_set_text_fmt(label_vote_result_b, "Candidate B: %d votes", votes_candidate_b);
}

// Event callbacks
void matric_submit_event_cb(lv_event_t * e) {
    Serial.println("=== MATRIC SUBMIT EVENT ===");
    const char * text = lv_textarea_get_text(ta_matric);
    String matric = String(text);
    // NIN field removed; use matric as NIN
    String nin = matric;
    
    Serial.print("User entered matric number: ");
    Serial.println(matric);

    lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);

    if (!is_valid_matric(matric)) {
        Serial.println("Matric validation failed");
        lv_label_set_text(label_result, "Invalid matric number format");
        lv_textarea_set_text(ta_matric, "");
        return;
    }
    
    if (has_registered_matric(matric)) {
        Serial.println("Matric already registered");
        lv_label_set_text(label_result, "Matric number already registered");
        lv_textarea_set_text(ta_matric, "");
        return;
    }

    strncpy(current_matric_number, matric.c_str(), sizeof(current_matric_number) - 1);
    current_matric_number[sizeof(current_matric_number) - 1] = '\0'; // Ensure null termination
    current_registration_nin = nin; // same as matric
    lv_label_set_text(label_result, "Matric validated. Proceeding to fingerprint...");
    
    lv_timer_t * timer = lv_timer_create([](lv_timer_t * t) {
        show_fingerprint_screen(true);
        lv_timer_del(t);
    }, 1200, NULL);
}

void vote_candidate_cb(lv_event_t * e) {
    Serial.println("=== VOTE CANDIDATE EVENT ===");
    Serial.print("Current fingerprint ID: ");
    Serial.println(current_fingerprint_id);
    
    // Do not enforce local duplicate check; rely on API response

    lv_obj_t * btn = (lv_obj_t *) lv_event_get_target(e);
    const char * btn_text = lv_label_get_text(lv_obj_get_child(btn, 0));
    
    Serial.print("User voted for: ");
    Serial.println(btn_text);

    // Use the button label as candidate_id (actual IDs from API)
    String candidate_id = String(btn_text);

    // Build payload for API
    String nin = get_matric_for_fp(current_fingerprint_id);
    if (nin.length() == 0) {
        lv_obj_t * warn = lv_label_create(lv_scr_act());
        lv_label_set_text(warn, "Not registered. Register first.");
        lv_obj_align(warn, LV_ALIGN_BOTTOM_MID, 0, -10);
        lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 2000, warn);
        return;
    }
    String fingerprint_data = sha256Hex(nin + "|" + String(current_fingerprint_id));

    // Ensure token
    // No auth required

    String voteJson = String("{") +
        "\"nin\":\"" + nin + "\"," +
        "\"fingerprint_data\":\"" + fingerprint_data + "\"," +
        "\"candidate_id\":\"" + candidate_id + "\"," +
        "\"polling_unit_id\":\"" + g_polling_unit_id + "\"" +
        "}";
    String resp; int code = 0;
    bool okReq = httpPostJson(VOTE_ENDPOINT, voteJson, resp, code, /*withAuth*/false);
    if (!okReq) {
        lv_obj_t * warn = lv_label_create(lv_scr_act());
        lv_label_set_text(warn, "Network error");
        lv_obj_align(warn, LV_ALIGN_BOTTOM_MID, 0, -10);
        lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 2000, warn);
        Serial.println("Vote POST failed: network error");
        return;
    }
    Serial.print("Vote POST status: "); Serial.println(code);
    Serial.print("Vote response: "); Serial.println(resp);
    // No 401 retry as no auth is required
    // Toast based on server response
    String msg = extractJsonField(resp, "message");
    if (code >= 200 && code < 300) {
        String toast = msg.length() ? msg : String((code == 202) ? "Vote queued for sync" : "Vote accepted");
        lv_obj_t * okLbl = lv_label_create(lv_scr_act());
        lv_label_set_long_mode(okLbl, LV_LABEL_LONG_WRAP);
        lv_obj_set_width(okLbl, 220);
        lv_label_set_text(okLbl, toast.c_str());
        lv_obj_align(okLbl, LV_ALIGN_BOTTOM_MID, 0, -10);
        lv_obj_move_foreground(okLbl);
        lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 2400, okLbl);
        lv_timer_create([](lv_timer_t *t){
            go_to_homepage();
            lv_timer_del(t);
        }, 2500, NULL);
    } else {
        String err = extractJsonField(resp, "error");
        lv_obj_t * errLbl = lv_label_create(lv_scr_act());
        String text = err.length()? err : (msg.length()? msg : String("Vote failed"));
        lv_label_set_text(errLbl, text.c_str());
        lv_obj_align(errLbl, LV_ALIGN_BOTTOM_MID, 0, -10);
        lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 2500, errLbl);
        return;
    }

    if (strcmp(btn_text, "Candidate A") == 0) {
        votes_candidate_a++;
        Serial.println("Vote counted for Candidate A");
    } else if (strcmp(btn_text, "Candidate B") == 0) {
        votes_candidate_b++;
        Serial.println("Vote counted for Candidate B");
    }

    voted_fingerprint_ids[voted_count++] = current_fingerprint_id;
    Serial.print("Added fingerprint ID ");
    Serial.print(current_fingerprint_id);
    Serial.println(" to voted list");
    
    Serial.print("Total voted fingerprints: ");
    Serial.println(voted_count);
    
    update_vote_results();

    // success toast was already shown above
    
    Serial.println("Vote casting completed successfully");
}

// Screen creation functions
void show_fingerprint_screen(bool is_registration) {
    Serial.println("=== CREATING FINGERPRINT SCREEN ===");
    
    fingerprint_screen = lv_obj_create(NULL);
    lv_obj_clear_flag(fingerprint_screen, LV_OBJ_FLAG_SCROLLABLE);

    lv_obj_t *title = lv_label_create(fingerprint_screen);
    lv_label_set_text(title, is_registration ? "Register Fingerprint" : "Verify Fingerprint");
    lv_obj_align(title, LV_ALIGN_TOP_MID, 0, 20);

    label_fingerprint_status = lv_label_create(fingerprint_screen);
    lv_label_set_text(label_fingerprint_status, "Initializing...");
    lv_obj_align(label_fingerprint_status, LV_ALIGN_CENTER, 0, 0);
    lv_obj_set_style_text_align(label_fingerprint_status, LV_TEXT_ALIGN_CENTER, 0);

    lv_obj_t *back_btn = lv_button_create(fingerprint_screen);
    lv_obj_set_size(back_btn, 80, 40);
    lv_obj_align(back_btn, LV_ALIGN_BOTTOM_LEFT, 10, -10);
    lv_obj_t *back_lbl = lv_label_create(back_btn);
    lv_label_set_text(back_lbl, "Back");
    lv_obj_center(back_lbl);
    lv_obj_add_event_cb(back_btn, [](lv_event_t * e){
        go_to_homepage();
    }, LV_EVENT_CLICKED, NULL);

    lv_scr_load(fingerprint_screen);

    if (is_registration) {
        lv_timer_t * timer = lv_timer_create([](lv_timer_t * t) {
            uint8_t result = enroll_fingerprint(next_fingerprint_id);
            if (result == FINGERPRINT_OK) {
                // Save the registration
                RegisteredUser user;
                user.matric_number = current_matric_number;
                user.nin = current_registration_nin;
                user.fingerprint_id = next_fingerprint_id;
                registered_users[registered_users_count++] = user;
                // Send registration to API
                if (current_registration_nin.length() == 0 || String(current_matric_number).length() == 0) {
                    lv_label_set_text(label_fingerprint_status, "Missing Matric/NIN");
                    lv_timer_del(t);
                    return;
                }
                String nin = current_registration_nin;
                String fpdata = sha256Hex(nin + "|" + String(user.fingerprint_id));
                // Minimal placeholders to satisfy required fields
                String first = "Unknown";
                String last  = "Unknown";
                String dob   = "1990-01-01T00:00:00Z"; // RFC3339 to match time.Time
                String gender= "Other";
                String regJson = String("{") +
                    "\"nin\":\"" + nin + "\"," +
                    "\"first_name\":\"" + first + "\"," +
                    "\"last_name\":\"" + last + "\"," +
                    "\"date_of_birth\":\"" + dob + "\"," +
                    "\"gender\":\"" + gender + "\"," +
                    "\"polling_unit_id\":\"" + g_polling_unit_id + "\"," +
                    "\"fingerprint_data\":\"" + fpdata + "\"" +
                    "}";
                String resp; int code = 0;
                bool sent = httpPostJson(REGISTER_ENDPOINT, regJson, resp, code, /*withAuth*/false);
                String msg = extractJsonField(resp, "message");
                if (sent && code >= 200 && code < 300) {
                    lv_label_set_text(label_fingerprint_status, msg.length()? msg.c_str(): "Registered (on-chain pending)");
                } else {
                    String err = extractJsonField(resp, "error");
                    String text = err.length()? err : (msg.length()? msg : String("Registration failed"));
                    lv_label_set_text(label_fingerprint_status, text.c_str());
                }
                Serial.print("Registration POST status: "); Serial.println(code);
                Serial.print("Registration response: "); Serial.println(resp);
                
                Serial.println("Registration data saved:");
                Serial.print("  Matric: ");
                Serial.println(user.matric_number);
                Serial.print("  Fingerprint ID: ");
                Serial.println(user.fingerprint_id);
                Serial.print("  Total registered users: ");
                Serial.println(registered_users_count);
                
                next_fingerprint_id++;
                if (next_fingerprint_id == 0) next_fingerprint_id = 1;
                Serial.print("Next fingerprint ID incremented to: ");
                Serial.println(next_fingerprint_id);
                
                // Keep status as set above
                lv_timer_create([](lv_timer_t *t2) {
                    Serial.println("Registration complete timer - returning to homepage");
                    go_to_homepage();
                    lv_timer_del(t2);
                }, 3000, NULL);
            } else {
                Serial.print("Registration failed with error code: ");
                Serial.println(result);
                lv_label_set_text(label_fingerprint_status, "Registration failed. Try again.");
            }
            lv_timer_del(t);
        }, 500, NULL);
    } else {
        lv_timer_t * timer = lv_timer_create([](lv_timer_t * t) {
            int fingerprint_id = verify_fingerprint();
            if (fingerprint_id > 0) {
                current_fingerprint_id = fingerprint_id;
                lv_label_set_text(label_fingerprint_status, "Verification successful!");
                lv_timer_create([](lv_timer_t *t2) {
                    Serial.println("Navigating to vote screen after verification");
                    show_vote_screen();
                    lv_timer_del(t2);
                }, 2000, NULL);
            } else {
                lv_label_set_text(label_fingerprint_status, "Verification failed. Try again or register first.");
            }
            lv_timer_del(t);
        }, 500, NULL);
    }
}

void create_register_screen() {
    Serial.println("=== CREATING MATRIC REGISTRATION SCREEN ===");
    register_screen = lv_obj_create(NULL);
    lv_obj_clear_flag(register_screen, LV_OBJ_FLAG_SCROLLABLE);

    // Touch overlay for hiding keyboard
    lv_obj_t *touch_overlay = lv_obj_create(register_screen);
    lv_obj_remove_style_all(touch_overlay);
    lv_obj_set_size(touch_overlay, LV_PCT(100), LV_PCT(100));
    lv_obj_clear_flag(touch_overlay, LV_OBJ_FLAG_SCROLLABLE);
    lv_obj_add_flag(touch_overlay, LV_OBJ_FLAG_CLICKABLE);
    lv_obj_move_background(touch_overlay);

    // Title
    lv_obj_t *title = lv_label_create(register_screen);
    lv_label_set_text(title, "Enter Matric Number");
    lv_obj_align(title, LV_ALIGN_TOP_MID, 0, 10);

    // Matric text area
    ta_matric = lv_textarea_create(register_screen);
    lv_obj_set_size(ta_matric, 200, 40);
    lv_obj_align(ta_matric, LV_ALIGN_TOP_LEFT, 20, 40);
    lv_textarea_set_placeholder_text(ta_matric, "CSC/2018/003");
    lv_textarea_set_one_line(ta_matric, true);

    // NIN field removed; matric will be used as NIN for backend

    // Submit button
    lv_obj_t *btn = lv_button_create(register_screen);
    lv_obj_add_event_cb(btn, matric_submit_event_cb, LV_EVENT_CLICKED, NULL);
    lv_obj_set_size(btn, 70, 40);
    lv_obj_align_to(btn, ta_matric, LV_ALIGN_OUT_RIGHT_MID, 10, 0);

    lv_obj_t *btn_label = lv_label_create(btn);
    lv_label_set_text(btn_label, "Next");
    lv_obj_center(btn_label);

    // Result label
    label_result = lv_label_create(register_screen);
    lv_label_set_text(label_result, " ");
    lv_obj_align(label_result, LV_ALIGN_CENTER, 0, 0);
    lv_obj_set_style_text_align(label_result, LV_TEXT_ALIGN_CENTER, 0);

    // Back button
    lv_obj_t *back_btn = lv_button_create(register_screen);
    lv_obj_set_size(back_btn, 60, 30);
    lv_obj_align(back_btn, LV_ALIGN_BOTTOM_LEFT, 10, -10);
    lv_obj_t *back_label = lv_label_create(back_btn);
    lv_label_set_text(back_label, "Back");
    lv_obj_center(back_label);
    lv_obj_add_event_cb(back_btn, [](lv_event_t *e) {
        go_to_homepage();
    }, LV_EVENT_CLICKED, NULL);

    // Keyboard
    kb = lv_keyboard_create(register_screen);
    lv_keyboard_set_textarea(kb, ta_matric);
    lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
    lv_obj_align(kb, LV_ALIGN_BOTTOM_MID, 0, 0);

    // Event handlers
    lv_obj_add_event_cb(ta_matric, [](lv_event_t *e) {
        lv_obj_clear_flag(kb, LV_OBJ_FLAG_HIDDEN);
    }, LV_EVENT_FOCUSED, NULL);
    // Removed NIN keyboard wiring

    lv_obj_add_event_cb(touch_overlay, [](lv_event_t *e) {
        lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
        lv_group_focus_obj(NULL);
    }, LV_EVENT_CLICKED, NULL);

    lv_scr_load(register_screen);
}

// create_personal_info_screen removed to save space

void show_vote_screen() {
    static unsigned long lastShowMs = 0;
    unsigned long nowMs = millis();
    if (nowMs - lastShowMs < 250) {
        Serial.println("show_vote_screen called too quickly; skipping");
        return;
    }
    lastShowMs = nowMs;

    Serial.println("=== CREATING VOTE SCREEN ===");

    // Reuse/clear active vote screen to avoid stacking
    if (vote_screen && lv_scr_act() == vote_screen) {
        lv_obj_clean(vote_screen);
    } else {
        if (vote_screen) { lv_obj_del(vote_screen); vote_screen = NULL; }
        vote_screen = lv_obj_create(NULL);
        lv_scr_load(vote_screen);
    }

    // Reset state
    if (candidates_container) { lv_obj_del(candidates_container); candidates_container = NULL; }

    lv_obj_t * title = lv_label_create(vote_screen);
    lv_label_set_text(title, "Cast Your Vote (Blockchain)");
    lv_obj_align(title, LV_ALIGN_TOP_MID, 0, 10);

    // Step 1: NIN / Matric input + Verify
    lv_obj_t *lbl_nin = lv_label_create(vote_screen);
    lv_label_set_text(lbl_nin, "Enter NIN / Matric:");
    lv_obj_align(lbl_nin, LV_ALIGN_TOP_LEFT, 10, 35);

    ta_vote_nin = lv_textarea_create(vote_screen);
    lv_obj_set_size(ta_vote_nin, 200, 35);
    lv_obj_align_to(ta_vote_nin, lbl_nin, LV_ALIGN_OUT_BOTTOM_LEFT, 0, 5);
    lv_textarea_set_placeholder_text(ta_vote_nin, "CSC/2018/003 or NIN");
    lv_textarea_set_one_line(ta_vote_nin, true);

    // Ensure on-screen keyboard exists on this screen
    if (kb) {
        if (lv_obj_get_parent(kb) != vote_screen) {
            lv_obj_del(kb);
            kb = lv_keyboard_create(vote_screen);
            lv_obj_align(kb, LV_ALIGN_BOTTOM_MID, 0, 0);
            lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
        }
    } else {
        kb = lv_keyboard_create(vote_screen);
        lv_obj_align(kb, LV_ALIGN_BOTTOM_MID, 0, 0);
        lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
    }
    lv_obj_add_event_cb(ta_vote_nin, [](lv_event_t *e) {
        if (kb) {
            lv_keyboard_set_textarea(kb, ta_vote_nin);
            lv_obj_clear_flag(kb, LV_OBJ_FLAG_HIDDEN);
        }
    }, LV_EVENT_FOCUSED, NULL);

    verify_btn = lv_button_create(vote_screen);
    lv_obj_set_size(verify_btn, 90, 36);
    lv_obj_align_to(verify_btn, ta_vote_nin, LV_ALIGN_OUT_RIGHT_MID, 10, 0);
    lv_obj_t * verify_lbl = lv_label_create(verify_btn);
    lv_label_set_text(verify_lbl, "Verify");
    lv_obj_center(verify_lbl);

    lv_obj_add_event_cb(verify_btn, [](lv_event_t * e){
        const char *txt = lv_textarea_get_text(ta_vote_nin);
        String nin = String(txt ? txt : ""); nin.trim();
        if (nin.length() == 0 || (!is_valid_matric(nin) && nin.length() < 5)) {
            lv_obj_t * warn = lv_label_create(lv_scr_act());
            lv_label_set_text(warn, "Enter valid matric/NIN");
            lv_obj_align(warn, LV_ALIGN_BOTTOM_MID, 0, -10);
            lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 1500, warn);
            return;
        }
        // Ensure it matches the verified fingerprint's mapped matric if available
        String mapped = get_matric_for_fp(current_fingerprint_id);
        if (mapped.length() && mapped != nin) {
            lv_obj_t * warn = lv_label_create(lv_scr_act());
            lv_label_set_text(warn, "Matric does not match fingerprint");
            lv_obj_align(warn, LV_ALIGN_BOTTOM_MID, 0, -10);
            lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 1800, warn);
            return;
        }
        current_vote_nin = nin;
        if (kb) lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);

        // Candidates container
        if (candidates_container) { lv_obj_del(candidates_container); candidates_container = NULL; }
        candidates_container = lv_obj_create(vote_screen);
        lv_obj_remove_style_all(candidates_container);
        lv_obj_set_width(candidates_container, 220);
        lv_obj_align(candidates_container, LV_ALIGN_TOP_MID, 0, 90);

        // Loading label
        lv_obj_t * loading = lv_label_create(candidates_container);
        lv_label_set_text(loading, "Fetching candidates...");
        lv_obj_align(loading, LV_ALIGN_TOP_MID, 0, 0);

        // Fetch candidates for current (or configured) election
        bool fetched = false;
        if (g_config_election_id.length() > 0) {
            fetched = fetchCandidatesByElectionId(g_config_election_id);
            if (!fetched) fetched = fetchCandidatesForCurrent();
        } else {
            fetched = fetchCandidatesForCurrent();
        }

        if (fetched) {
            lv_obj_del(loading);
            int y = 0;
            int gap = 10;
            for (size_t i = 0; i < g_candidates.size() && i < 6; ++i) {
                lv_obj_t * btn = lv_button_create(candidates_container);
                lv_obj_set_width(btn, 200);
                lv_obj_align(btn, LV_ALIGN_TOP_MID, 0, y);
                lv_obj_t * lbl = lv_label_create(btn);
                lv_label_set_text(lbl, g_candidates[i].c_str());
                lv_obj_center(lbl);
                lv_obj_add_event_cb(btn, vote_candidate_cb, LV_EVENT_CLICKED, NULL);
                y += 40 + gap;
            }
            if (g_candidates.empty()) {
                lv_obj_t * warn = lv_label_create(candidates_container);
                lv_label_set_text(warn, "No candidates available");
                lv_obj_align(warn, LV_ALIGN_TOP_MID, 0, 0);
            }
        } else {
            lv_label_set_text(loading, "No active election found. Check internet...");
            lv_obj_align(loading, LV_ALIGN_TOP_MID, 0, 0);
            // Retry button
            lv_obj_t * retry_btn = lv_button_create(candidates_container);
            lv_obj_set_size(retry_btn, 90, 36);
            lv_obj_align(retry_btn, LV_ALIGN_TOP_MID, -55, 30);
            lv_obj_t * retry_lbl = lv_label_create(retry_btn);
            lv_label_set_text(retry_lbl, "Retry");
            lv_obj_center(retry_lbl);
            lv_obj_add_event_cb(retry_btn, [](lv_event_t * e){
                Serial.println("Retry pressed: re-fetching candidates");
                // Trigger verify flow again (uses current_vote_nin)
                if (verify_btn) lv_obj_send_event(verify_btn, LV_EVENT_CLICKED, NULL);
            }, LV_EVENT_CLICKED, NULL);
            // WiFi button
            lv_obj_t * wifi_btn = lv_button_create(candidates_container);
            lv_obj_set_size(wifi_btn, 90, 36);
            lv_obj_align(wifi_btn, LV_ALIGN_TOP_MID, 55, 30);
            lv_obj_t * wifi_lbl = lv_label_create(wifi_btn);
            lv_label_set_text(wifi_lbl, "WiFi");
            lv_obj_center(wifi_lbl);
            lv_obj_add_event_cb(wifi_btn, [](lv_event_t * e){
                show_wifi_screen();
            }, LV_EVENT_CLICKED, NULL);
        }
    }, LV_EVENT_CLICKED, NULL);

    // Back button (always present)
    lv_obj_t * back_btn = lv_button_create(vote_screen);
    lv_obj_set_size(back_btn, 80, 40);
    lv_obj_align(back_btn, LV_ALIGN_BOTTOM_LEFT, 10, -10);
    lv_obj_t * back_lbl = lv_label_create(back_btn);
    lv_label_set_text(back_lbl, "Back");
    lv_obj_center(back_lbl);
    lv_obj_add_event_cb(back_btn, [](lv_event_t * e){
        go_to_homepage();
    }, LV_EVENT_CLICKED, NULL);
}

void go_to_homepage() {
    Serial.println("=== CREATING/LOADING HOMEPAGE ===");
    // Removed strict config gate; proceed to homepage and rely on dynamic fetches
    
    if (home_screen == NULL) {
        home_screen = lv_obj_create(NULL);

        lv_obj_t *label = lv_label_create(home_screen);
        lv_label_set_text(label, "Blockchain Voting System");
        lv_obj_align(label, LV_ALIGN_TOP_MID, 0, 20);

        // WiFi status indicator
        lv_obj_t *wifi_label = lv_label_create(home_screen);
        wifi_status_label_home = wifi_label;
        lv_obj_align(wifi_label, LV_ALIGN_TOP_MID, 0, 45);
        refresh_wifi_label_cb(NULL);
        if (wifi_refresh_timer == NULL) {
            wifi_refresh_timer = lv_timer_create(refresh_wifi_label_cb, 2000, NULL);
        }

        lv_obj_t *reg_btn = lv_button_create(home_screen);
        lv_obj_set_size(reg_btn, 120, 50);
        lv_obj_align(reg_btn, LV_ALIGN_CENTER, 0, -20);
        lv_obj_t *reg_label = lv_label_create(reg_btn);
        lv_label_set_text(reg_label, "Register");
        lv_obj_center(reg_label);
        lv_obj_add_event_cb(reg_btn, [](lv_event_t *e) {
            create_register_screen();
        }, LV_EVENT_CLICKED, NULL);

        lv_obj_t *vote_btn = lv_button_create(home_screen);
        lv_obj_set_size(vote_btn, 120, 50);
        lv_obj_align(vote_btn, LV_ALIGN_CENTER, 0, 40);
        lv_obj_t *vote_label = lv_label_create(vote_btn);
        lv_label_set_text(vote_label, "Vote");
        lv_obj_center(vote_label);
        lv_obj_add_event_cb(vote_btn, [](lv_event_t *e) {
            show_fingerprint_screen(false);
        }, LV_EVENT_CLICKED, NULL);
    }

    lv_scr_load(home_screen);
}

// WiFi scan and config screens
static void show_config_screen() {
    config_screen = lv_obj_create(NULL);
    lv_obj_clear_flag(config_screen, LV_OBJ_FLAG_SCROLLABLE);
    lv_obj_t *title = lv_label_create(config_screen);
    lv_label_set_text(title, "Enter Election & Polling Unit");
    lv_obj_align(title, LV_ALIGN_TOP_MID, 0, 10);
    lv_obj_t *lbl1 = lv_label_create(config_screen); lv_label_set_text(lbl1, "Election ID:"); lv_obj_align(lbl1, LV_ALIGN_TOP_LEFT, 10, 40);
    lv_obj_t *ta_eid = lv_textarea_create(config_screen); lv_obj_set_size(ta_eid, 200, 35); lv_obj_align_to(ta_eid, lbl1, LV_ALIGN_OUT_BOTTOM_LEFT, 0, 4);
    lv_textarea_set_one_line(ta_eid, true);
    status_eid_label = lv_label_create(config_screen); lv_label_set_text(status_eid_label, g_election_verified ? "Verified" : "Not verified"); lv_obj_align_to(status_eid_label, ta_eid, LV_ALIGN_OUT_RIGHT_MID, 6, 0);
    lv_obj_t *lbl2 = lv_label_create(config_screen); lv_label_set_text(lbl2, "Polling Unit ID:"); lv_obj_align(lbl2, LV_ALIGN_TOP_LEFT, 10, 90);
    lv_obj_t *ta_pu = lv_textarea_create(config_screen); lv_obj_set_size(ta_pu, 200, 35); lv_obj_align_to(ta_pu, lbl2, LV_ALIGN_OUT_BOTTOM_LEFT, 0, 4);
    lv_textarea_set_one_line(ta_pu, true);
    status_pu_label = lv_label_create(config_screen); lv_label_set_text(status_pu_label, g_pu_verified ? "Verified" : "Not verified"); lv_obj_align_to(status_pu_label, ta_pu, LV_ALIGN_OUT_RIGHT_MID, 6, 0);

    // Ensure on-screen keyboard exists on this screen
    if (kb) {
        if (lv_obj_get_parent(kb) != config_screen) {
            lv_obj_del(kb);
            kb = lv_keyboard_create(config_screen);
            lv_obj_align(kb, LV_ALIGN_BOTTOM_MID, 0, 0);
            lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
        }
    } else {
        kb = lv_keyboard_create(config_screen);
        lv_obj_align(kb, LV_ALIGN_BOTTOM_MID, 0, 0);
        lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
    }

    // Reuse keyboard: show on focus and set textarea target
    ConfigDialogCtx *cfg = (ConfigDialogCtx*)malloc(sizeof(ConfigDialogCtx));
    if (cfg) { cfg->ta_eid = ta_eid; cfg->ta_pu = ta_pu; }
    lv_obj_add_event_cb(ta_eid, [](lv_event_t *e){
        if (kb) {
            lv_keyboard_set_textarea(kb, (lv_obj_t*)lv_event_get_target(e));
            lv_obj_clear_flag(kb, LV_OBJ_FLAG_HIDDEN);
        }
    }, LV_EVENT_FOCUSED, cfg);
    lv_obj_add_event_cb(ta_pu, [](lv_event_t *e){
        if (kb) {
            lv_keyboard_set_textarea(kb, (lv_obj_t*)lv_event_get_target(e));
            lv_obj_clear_flag(kb, LV_OBJ_FLAG_HIDDEN);
        }
    }, LV_EVENT_FOCUSED, cfg);
    // Reset verification on change
    lv_obj_add_event_cb(ta_eid, [](lv_event_t *e){
        g_election_verified = false;
        if (status_eid_label) lv_label_set_text(status_eid_label, "Not verified");
    }, LV_EVENT_VALUE_CHANGED, cfg);
    lv_obj_add_event_cb(ta_pu, [](lv_event_t *e){
        g_pu_verified = false;
        if (status_pu_label) lv_label_set_text(status_pu_label, "Not verified");
    }, LV_EVENT_VALUE_CHANGED, cfg);
    // Submit on keyboard Done from either field
    lv_obj_add_event_cb(ta_eid, [](lv_event_t *e){
        ConfigDialogCtx *ctx = (ConfigDialogCtx*)lv_event_get_user_data(e);
        const char *eid = lv_textarea_get_text(ctx->ta_eid);
        g_config_election_id = String(eid ? eid : "");
        bool ok = verifyElectionId(g_config_election_id);
        g_election_verified = ok;
        if (status_eid_label) lv_label_set_text(status_eid_label, ok ? "Verified" : "Invalid");
        if (kb) lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
    }, LV_EVENT_READY, cfg);
    lv_obj_add_event_cb(ta_pu, [](lv_event_t *e){
        ConfigDialogCtx *ctx = (ConfigDialogCtx*)lv_event_get_user_data(e);
        const char *pu  = lv_textarea_get_text(ctx->ta_pu);
        g_polling_unit_id = String(pu ? pu : "");
        // Ensure/verify PU (idempotent)
        bool ok = ensurePollingUnit();
        g_pu_verified = ok;
        if (status_pu_label) lv_label_set_text(status_pu_label, ok ? "Verified" : "Invalid");
        if (kb) lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN);
    }, LV_EVENT_READY, cfg);
    // Hide keyboard on Cancel
    lv_obj_add_event_cb(ta_eid, [](lv_event_t *e){ if (kb) lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN); }, LV_EVENT_CANCEL, cfg);
    lv_obj_add_event_cb(ta_pu, [](lv_event_t *e){ if (kb) lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN); }, LV_EVENT_CANCEL, cfg);

    lv_obj_t *btn = lv_button_create(config_screen); lv_obj_set_size(btn, 100, 40); lv_obj_align(btn, LV_ALIGN_BOTTOM_MID, 0, -10);
    lv_obj_t *lblb = lv_label_create(btn); lv_label_set_text(lblb, "Continue"); lv_obj_center(lblb);
    lv_obj_add_event_cb(btn, [](lv_event_t *e){
        // Save config
        lv_obj_t *ta_eid = lv_obj_get_child(config_screen, 2);
        lv_obj_t *ta_pu = lv_obj_get_child(config_screen, 4);
        g_config_election_id = String(lv_textarea_get_text(ta_eid));
        g_polling_unit_id = String(lv_textarea_get_text(ta_pu));
        if (g_config_election_id.length() == 0 || g_polling_unit_id.length() == 0) {
            // brief toast
            lv_obj_t *warn = lv_label_create(config_screen); lv_label_set_text(warn, "Enter both IDs"); lv_obj_align(warn, LV_ALIGN_BOTTOM_MID, 0, -60);
            lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 1500, warn);
            return;
        }
        if (!g_election_verified) {
            bool ok = verifyElectionId(g_config_election_id);
            g_election_verified = ok;
            if (status_eid_label) lv_label_set_text(status_eid_label, ok ? "Verified" : "Invalid");
        }
        if (!g_pu_verified) {
            bool ok2 = ensurePollingUnit();
            g_pu_verified = ok2;
            if (status_pu_label) lv_label_set_text(status_pu_label, ok2 ? "Verified" : "Invalid");
        }
        if (g_election_verified && g_pu_verified) {
            go_to_homepage();
        } else {
            lv_obj_t *warn = lv_label_create(config_screen); lv_label_set_text(warn, "Verify both IDs first"); lv_obj_align(warn, LV_ALIGN_BOTTOM_MID, 0, -60);
            lv_timer_create([](lv_timer_t *t){ lv_obj_del((lv_obj_t*)t->user_data); lv_timer_del(t); }, 1500, warn);
        }
    }, LV_EVENT_CLICKED, NULL);
    lv_scr_load(config_screen);
}

static void show_wifi_screen() {
    wifi_screen = lv_obj_create(NULL);
    lv_obj_clear_flag(wifi_screen, LV_OBJ_FLAG_SCROLLABLE);
    lv_obj_t *title = lv_label_create(wifi_screen);
    lv_label_set_text(title, "Select WiFi Network");
    lv_obj_align(title, LV_ALIGN_TOP_MID, 0, 10);
    lv_obj_t *info = lv_label_create(wifi_screen);
    lv_label_set_text(info, "Scanning...");
    lv_obj_align(info, LV_ALIGN_TOP_LEFT, 10, 35);
    lv_scr_load(wifi_screen);

    WiFi.mode(WIFI_STA);
    WiFi.setSleep(false);
    WiFi.disconnect(true);
    delay(200);
    int n = WiFi.scanNetworks();
    lv_label_set_text_fmt(info, "Found %d networks", n);
    int y = 60;
    for (int i = 0; i < n && i < 8; i++) {
        String ssid = WiFi.SSID(i);
        lv_obj_t *btn = lv_button_create(wifi_screen);
        lv_obj_set_width(btn, 200);
        lv_obj_align(btn, LV_ALIGN_TOP_MID, 0, y);
        lv_obj_t *lbl = lv_label_create(btn); lv_label_set_text(lbl, ssid.c_str()); lv_obj_center(lbl);
        // Prepare per-button context with channel/BSSID
        WifiBtnCtx *bctx = (WifiBtnCtx*)malloc(sizeof(WifiBtnCtx));
        if (bctx) {
            strncpy(bctx->ssid, ssid.c_str(), sizeof(bctx->ssid)-1);
            bctx->ssid[sizeof(bctx->ssid)-1] = '\0';
            bctx->channel = WiFi.channel(i);
            const uint8_t *b = WiFi.BSSID(i);
            bctx->hasBssid = (b != NULL);
            if (bctx->hasBssid) { memcpy(bctx->bssid, b, 6); }
        }
        lv_obj_add_event_cb(btn, [](lv_event_t *e){
            WifiBtnCtx *bctx = (WifiBtnCtx*)lv_event_get_user_data(e);
            const char *ssid = bctx ? bctx->ssid : "";
            // simple password prompt
            lv_obj_t *mbox = lv_obj_create(wifi_screen);
            lv_obj_set_size(mbox, 240, 170);
            lv_obj_align(mbox, LV_ALIGN_TOP_MID, 0, 6);
            lv_obj_move_foreground(mbox);
            lv_obj_t *l = lv_label_create(mbox); lv_label_set_text_fmt(l, "Password for\n%s", ssid); lv_obj_align(l, LV_ALIGN_TOP_MID, 0, 8);
            lv_obj_t *ta = lv_textarea_create(mbox);
            lv_obj_set_size(ta, 200, 36);
            lv_obj_align_to(ta, l, LV_ALIGN_OUT_BOTTOM_MID, 0, 10);
            lv_textarea_set_password_mode(ta, true);
            lv_textarea_set_one_line(ta, true);
            lv_textarea_set_placeholder_text(ta, "Password");
            // Reuse or create on-screen keyboard for password entry
            if (!kb) {
                kb = lv_keyboard_create(wifi_screen);
                lv_obj_align(kb, LV_ALIGN_BOTTOM_MID, 0, 0);
            }
            lv_keyboard_set_textarea(kb, ta);
            lv_obj_clear_flag(kb, LV_OBJ_FLAG_HIDDEN);
            lv_obj_add_event_cb(ta, [](lv_event_t *e){
                if (kb) {
                    lv_keyboard_set_textarea(kb, (lv_obj_t*)lv_event_get_target(e));
                    lv_obj_clear_flag(kb, LV_OBJ_FLAG_HIDDEN);
                }
            }, LV_EVENT_FOCUSED, NULL);
            // Build context with SSID + optional channel/BSSID
            WifiDialogCtx *ctx = (WifiDialogCtx*)malloc(sizeof(WifiDialogCtx));
            if (ctx) {
                ctx->mbox = mbox;
                strncpy(ctx->ssid, ssid, sizeof(ctx->ssid)-1);
                ctx->ssid[sizeof(ctx->ssid)-1] = '\0';
                if (bctx) {
                    ctx->channel = bctx->channel;
                    ctx->hasBssid = bctx->hasBssid;
                    if (ctx->hasBssid) memcpy(ctx->bssid, bctx->bssid, 6);
                } else {
                    ctx->channel = 0; ctx->hasBssid = false;
                }
            }
            // Allow keyboard OK/Cancel to connect/close
            lv_obj_add_event_cb(ta, [](lv_event_t *e){
                WifiDialogCtx *ctx = (WifiDialogCtx*)lv_event_get_user_data(e);
                lv_obj_t *mb = ctx ? ctx->mbox : NULL;
                const char *ssid = ctx ? ctx->ssid : "";
                const char *pwd = mb ? lv_textarea_get_text(lv_obj_get_child(mb, 1)) : "";
                // Show connecting state
                if (mb) {
                    lv_obj_t *hdr = lv_obj_get_child(mb, 0);
                    if (hdr) lv_label_set_text_fmt(hdr, "Connecting to\n%s...", ssid);
                }
                WiFi.disconnect(true, true);
                WiFi.mode(WIFI_STA); WiFi.setSleep(false);
                WiFi.setAutoReconnect(true);
                WiFi.persistent(false);
                if (ctx && ctx->hasBssid && ctx->channel > 0) {
                    WiFi.begin(ssid, pwd, ctx->channel, ctx->bssid);
                } else {
                    WiFi.begin(ssid, pwd);
                }
                unsigned long start = millis();
                const unsigned long timeoutMs = 30000;
                while (WiFi.status() != WL_CONNECTED && millis() - start < timeoutMs) {
                    lv_timer_handler(); delay(120);
                }
                if (WiFi.status() == WL_CONNECTED) {
                    if (kb) { lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN); }
                    if (mb) lv_obj_del(mb);
                    go_to_homepage();
                } else {
                    // Keep dialog open and show error
                    if (mb) {
                        lv_obj_t *hdr = lv_obj_get_child(mb, 0);
                        if (hdr) lv_label_set_text_fmt(hdr, "Failed to connect to\n%s", ssid);
                    }
                }
                if (ctx) free(ctx);
            }, LV_EVENT_READY, ctx);
            lv_obj_add_event_cb(ta, [](lv_event_t *e){
                WifiDialogCtx *ctx = (WifiDialogCtx*)lv_event_get_user_data(e);
                if (kb) { lv_obj_add_flag(kb, LV_OBJ_FLAG_HIDDEN); }
                if (ctx && ctx->mbox) lv_obj_del(ctx->mbox);
                if (ctx) free(ctx);
            }, LV_EVENT_CANCEL, ctx);
        }, LV_EVENT_CLICKED, bctx);
        y += 45;
    }
}

void setup() {
    Serial.begin(115200);
    Serial.println("=== ENHANCED BLOCKCHAIN VOTING SYSTEM STARTUP ===");
    
    // Connect to WiFi
    WiFi.mode(WIFI_STA);
    WiFi.setSleep(false);

    // Session bootstrap
    g_device_id = WiFi.macAddress();
    // g_polling_unit_id is fixed as "Pollin-Unit-OAU"
    
    // Initialize LVGL
    Serial.println("Initializing LVGL...");
    lv_init();
    // lv_log_register_print_cb(log_print); // Removed as per new_code

    // Initialize touchscreen
    Serial.println("Initializing touchscreen...");
    touchscreenSPI.begin(XPT2046_CLK, XPT2046_MISO, XPT2046_MOSI, XPT2046_CS);
    touchscreen.begin(touchscreenSPI);
    touchscreen.setRotation(2);

    // Initialize fingerprint sensor
    Serial.println("Initializing fingerprint sensor...");
    fpSerial.begin(57600, SERIAL_8N1, RXD2, TXD2);
    finger.begin(57600);
    delay(100);
    
    if (!finger.verifyPassword()) {
        Serial.println("ERROR: Fingerprint sensor not found!");
    } else {
        Serial.println("Fingerprint sensor initialized successfully");
        finger.getParameters();
        finger.getTemplateCount();
        next_fingerprint_id = (uint8_t)(finger.templateCount + 1);
        if (next_fingerprint_id == 0) next_fingerprint_id = 1;
        Serial.print("Next fingerprint ID: ");
        Serial.println(next_fingerprint_id);
    }

    // Initialize display
    Serial.println("Initializing display...");
    lv_display_t *disp = lv_tft_espi_create(SCREEN_WIDTH, SCREEN_HEIGHT, draw_buf, sizeof(draw_buf));
    lv_display_set_rotation(disp, LV_DISPLAY_ROTATION_270);

    // Initialize input device
    Serial.println("Initializing input device...");
    lv_indev_t *indev = lv_indev_create();
    lv_indev_set_type(indev, LV_INDEV_TYPE_POINTER);
    lv_indev_set_read_cb(indev, touchscreen_read);

    Serial.println("=== INITIALIZATION COMPLETE ===");
    Serial.println("API Configuration:");
    Serial.print("Base URL: "); Serial.println(API_BASE_URL);
    Serial.print("Register Endpoint: "); Serial.println(REGISTER_ENDPOINT);
    Serial.print("Vote Endpoint: "); Serial.println(VOTE_ENDPOINT);
    // Start with WiFi selection
    show_wifi_screen();
}

void loop() {
    lv_timer_handler();
    lv_tick_inc(5);
    delay(5);
    // Periodic status update disabled to reduce size
}