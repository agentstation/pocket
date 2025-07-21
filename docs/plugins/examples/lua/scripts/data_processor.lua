-- Data processing script example
-- This script demonstrates various Lua features available in Pocket

-- Helper function to calculate statistics
local function calculate_stats(numbers)
    local sum = 0
    local min = math.huge
    local max = -math.huge
    
    for _, n in ipairs(numbers) do
        sum = sum + n
        if n < min then min = n end
        if n > max then max = n end
    end
    
    local avg = sum / #numbers
    
    return {
        sum = sum,
        average = avg,
        min = min,
        max = max,
        count = #numbers
    }
end

-- Main processing logic
local result = {
    timestamp = os.time(),
    original_input = input
}

-- Check if input has data to process
if input and input.data then
    -- Process numerical data
    if input.data.numbers then
        result.number_stats = calculate_stats(input.data.numbers)
    end
    
    -- Process text data
    if input.data.text then
        local text = input.data.text
        result.text_analysis = {
            original = text,
            uppercase = string.upper(text),
            lowercase = string.lower(text),
            trimmed = str_trim(text),
            words = str_split(str_trim(text), " "),
            contains_hello = str_contains(text, "hello"),
            replaced = str_replace(text, "world", "Lua")
        }
        result.text_analysis.word_count = #result.text_analysis.words
    end
    
    -- Process JSON data
    if input.data.json_string then
        local decoded = json_decode(input.data.json_string)
        result.json_processed = {
            decoded = decoded,
            re_encoded = json_encode(decoded),
            type = type(decoded)
        }
    end
end

-- Add processing metadata
result.processed_at = os.date("%Y-%m-%d %H:%M:%S")
result.processor_version = "1.0"

-- Return the processed result
return result