use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::slice;
use std::str;

// Plugin metadata types
#[derive(Serialize, Deserialize)]
struct Metadata {
    name: String,
    version: String,
    description: String,
    author: String,
    license: String,
    runtime: String,
    binary: String,
    nodes: Vec<NodeDefinition>,
    permissions: Permissions,
    requirements: Requirements,
}

#[derive(Serialize, Deserialize)]
struct NodeDefinition {
    #[serde(rename = "type")]
    node_type: String,
    category: String,
    description: String,
    #[serde(rename = "configSchema")]
    config_schema: Option<serde_json::Value>,
    #[serde(rename = "inputSchema")]
    input_schema: Option<serde_json::Value>,
    #[serde(rename = "outputSchema")]
    output_schema: Option<serde_json::Value>,
}

#[derive(Serialize, Deserialize)]
struct Permissions {
    memory: String,
    timeout: u64,
}

#[derive(Serialize, Deserialize)]
struct Requirements {
    pocket: String,
}

// Request/Response types
#[derive(Serialize, Deserialize)]
struct Request {
    node: String,
    function: String,
    config: Option<serde_json::Value>,
    input: Option<serde_json::Value>,
}

#[derive(Serialize, Deserialize)]
struct Response {
    success: bool,
    output: Option<serde_json::Value>,
    error: Option<String>,
    next: Option<String>,
}

// Word counter specific types
#[derive(Serialize, Deserialize)]
struct WordCounterInput {
    text: String,
    #[serde(default)]
    case_sensitive: bool,
}

#[derive(Serialize, Deserialize)]
struct WordCounterOutput {
    total_words: usize,
    unique_words: usize,
    word_frequencies: HashMap<String, usize>,
    average_word_length: f64,
    longest_word: String,
    shortest_word: String,
}

#[derive(Serialize, Deserialize)]
struct WordCounterConfig {
    #[serde(default = "default_min_word_length")]
    min_word_length: usize,
    #[serde(default = "default_stop_words")]
    stop_words: Vec<String>,
}

fn default_min_word_length() -> usize {
    1
}

fn default_stop_words() -> Vec<String> {
    vec![
        "a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
        "has", "he", "in", "is", "it", "its", "of", "on", "that", "the",
        "to", "was", "will", "with"
    ].into_iter().map(String::from).collect()
}

// Memory management functions
#[no_mangle]
pub extern "C" fn alloc(size: usize) -> *mut u8 {
    let mut buf = Vec::with_capacity(size);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

#[no_mangle]
pub extern "C" fn dealloc(ptr: *mut u8, size: usize) {
    unsafe {
        let _ = Vec::from_raw_parts(ptr, size, size);
    }
}

// Plugin metadata export
#[no_mangle]
pub extern "C" fn metadata(ptr: *mut u8, len: usize) -> usize {
    let metadata = Metadata {
        name: "word-counter".to_string(),
        version: "1.0.0".to_string(),
        description: "Word counting and analysis plugin for Pocket".to_string(),
        author: "Pocket Team".to_string(),
        license: "MIT".to_string(),
        runtime: "wasm".to_string(),
        binary: "plugin.wasm".to_string(),
        nodes: vec![
            NodeDefinition {
                node_type: "word-count".to_string(),
                category: "text".to_string(),
                description: "Count words and analyze text statistics".to_string(),
                config_schema: Some(serde_json::json!({
                    "type": "object",
                    "properties": {
                        "min_word_length": {
                            "type": "integer",
                            "default": 1,
                            "minimum": 1,
                            "description": "Minimum word length to count"
                        },
                        "stop_words": {
                            "type": "array",
                            "items": {"type": "string"},
                            "default": ["a", "an", "and", "are", "as", "at", "be", "by", "for", "from",
                                       "has", "he", "in", "is", "it", "its", "of", "on", "that", "the",
                                       "to", "was", "will", "with"],
                            "description": "Words to exclude from counting"
                        }
                    }
                })),
                input_schema: Some(serde_json::json!({
                    "type": "object",
                    "properties": {
                        "text": {
                            "type": "string",
                            "minLength": 1,
                            "description": "Text to analyze"
                        },
                        "case_sensitive": {
                            "type": "boolean",
                            "default": false,
                            "description": "Whether to treat words case-sensitively"
                        }
                    },
                    "required": ["text"]
                })),
                output_schema: Some(serde_json::json!({
                    "type": "object",
                    "properties": {
                        "total_words": {"type": "integer"},
                        "unique_words": {"type": "integer"},
                        "word_frequencies": {
                            "type": "object",
                            "additionalProperties": {"type": "integer"}
                        },
                        "average_word_length": {"type": "number"},
                        "longest_word": {"type": "string"},
                        "shortest_word": {"type": "string"}
                    },
                    "required": ["total_words", "unique_words", "word_frequencies", 
                               "average_word_length", "longest_word", "shortest_word"]
                })),
            }
        ],
        permissions: Permissions {
            memory: "5MB".to_string(),
            timeout: 3000,
        },
        requirements: Requirements {
            pocket: ">=1.0.0".to_string(),
        },
    };

    let json = serde_json::to_string(&metadata).unwrap();
    let bytes = json.as_bytes();
    
    unsafe {
        std::ptr::copy(bytes.as_ptr(), ptr, bytes.len().min(len));
    }
    
    bytes.len()
}

// Main call function
#[no_mangle]
pub extern "C" fn call(ptr: *const u8, len: usize, out_ptr: *mut u8, out_len: usize) -> usize {
    let input = unsafe { slice::from_raw_parts(ptr, len) };
    let input_str = match str::from_utf8(input) {
        Ok(s) => s,
        Err(_) => {
            let error_response = Response {
                success: false,
                output: None,
                error: Some("Invalid UTF-8 input".to_string()),
                next: None,
            };
            let output = serde_json::to_string(&error_response).unwrap();
            let output_bytes = output.as_bytes();
            unsafe {
                std::ptr::copy(output_bytes.as_ptr(), out_ptr, output_bytes.len().min(out_len));
            }
            return output_bytes.len();
        }
    };

    let request: Request = match serde_json::from_str(input_str) {
        Ok(r) => r,
        Err(e) => {
            let error_response = Response {
                success: false,
                output: None,
                error: Some(format!("Failed to parse request: {}", e)),
                next: None,
            };
            let output = serde_json::to_string(&error_response).unwrap();
            let output_bytes = output.as_bytes();
            unsafe {
                std::ptr::copy(output_bytes.as_ptr(), out_ptr, output_bytes.len().min(out_len));
            }
            return output_bytes.len();
        }
    };

    let response = match request.function.as_str() {
        "prep" => handle_prep(&request),
        "exec" => handle_exec(&request),
        "post" => handle_post(&request),
        _ => Response {
            success: false,
            output: None,
            error: Some(format!("Unknown function: {}", request.function)),
            next: None,
        },
    };

    let output = serde_json::to_string(&response).unwrap();
    let output_bytes = output.as_bytes();
    
    unsafe {
        std::ptr::copy(output_bytes.as_ptr(), out_ptr, output_bytes.len().min(out_len));
    }
    
    output_bytes.len()
}

fn handle_prep(request: &Request) -> Response {
    let input: WordCounterInput = match request.input.as_ref() {
        Some(i) => match serde_json::from_value(i.clone()) {
            Ok(inp) => inp,
            Err(e) => return Response {
                success: false,
                output: None,
                error: Some(format!("Failed to parse input: {}", e)),
                next: None,
            },
        },
        None => return Response {
            success: false,
            output: None,
            error: Some("No input provided".to_string()),
            next: None,
        },
    };

    // Clean and prepare text
    let cleaned_text = input.text
        .chars()
        .map(|c| if c.is_alphanumeric() || c.is_whitespace() { c } else { ' ' })
        .collect::<String>();

    let prep_result = serde_json::json!({
        "original_text": input.text,
        "cleaned_text": cleaned_text,
        "case_sensitive": input.case_sensitive,
    });

    Response {
        success: true,
        output: Some(prep_result),
        error: None,
        next: None,
    }
}

fn handle_exec(request: &Request) -> Response {
    let prep_data = match request.input.as_ref() {
        Some(d) => d,
        None => return Response {
            success: false,
            output: None,
            error: Some("No prep data provided".to_string()),
            next: None,
        },
    };

    let cleaned_text = prep_data["cleaned_text"].as_str().unwrap_or("");
    let case_sensitive = prep_data["case_sensitive"].as_bool().unwrap_or(false);

    let config: WordCounterConfig = request.config.as_ref()
        .and_then(|c| serde_json::from_value(c.clone()).ok())
        .unwrap_or_else(|| WordCounterConfig {
            min_word_length: default_min_word_length(),
            stop_words: default_stop_words(),
        });

    // Split into words
    let words: Vec<String> = cleaned_text
        .split_whitespace()
        .filter(|w| w.len() >= config.min_word_length)
        .map(|w| if case_sensitive { w.to_string() } else { w.to_lowercase() })
        .filter(|w| !config.stop_words.contains(w))
        .collect();

    if words.is_empty() {
        return Response {
            success: true,
            output: Some(serde_json::json!(WordCounterOutput {
                total_words: 0,
                unique_words: 0,
                word_frequencies: HashMap::new(),
                average_word_length: 0.0,
                longest_word: String::new(),
                shortest_word: String::new(),
            })),
            error: None,
            next: None,
        };
    }

    // Count word frequencies
    let mut word_frequencies = HashMap::new();
    let mut total_length = 0;
    let mut longest_word = &words[0];
    let mut shortest_word = &words[0];

    for word in &words {
        *word_frequencies.entry(word.clone()).or_insert(0) += 1;
        total_length += word.len();
        
        if word.len() > longest_word.len() {
            longest_word = word;
        }
        if word.len() < shortest_word.len() {
            shortest_word = word;
        }
    }

    let output = WordCounterOutput {
        total_words: words.len(),
        unique_words: word_frequencies.len(),
        word_frequencies,
        average_word_length: total_length as f64 / words.len() as f64,
        longest_word: longest_word.clone(),
        shortest_word: shortest_word.clone(),
    };

    Response {
        success: true,
        output: Some(serde_json::to_value(output).unwrap()),
        error: None,
        next: None,
    }
}

fn handle_post(request: &Request) -> Response {
    let exec_result = match request.input.as_ref() {
        Some(r) => r,
        None => return Response {
            success: false,
            output: None,
            error: Some("No exec result provided".to_string()),
            next: None,
        },
    };

    let total_words = exec_result["total_words"].as_u64().unwrap_or(0);
    
    // Route based on word count
    let next = if total_words == 0 {
        "empty"
    } else if total_words < 100 {
        "short"
    } else if total_words < 1000 {
        "medium"
    } else {
        "long"
    };

    Response {
        success: true,
        output: Some(exec_result.clone()),
        error: None,
        next: Some(next.to_string()),
    }
}