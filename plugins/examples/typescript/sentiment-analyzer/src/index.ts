import { Plugin, PluginNode, initializePlugin } from '@pocket/plugin-sdk';

// Define input and output types
interface SentimentInput {
  text: string;
  lang?: string;
}

interface SentimentOutput {
  sentiment: 'positive' | 'negative' | 'neutral';
  score: number;
  confidence: number;
  keywords: string[];
}

interface SentimentConfig {
  threshold?: number;
  languages?: string[];
}

// Sentiment analyzer node
class SentimentAnalyzerNode extends PluginNode<SentimentInput, SentimentOutput, SentimentConfig> {
  readonly type = 'sentiment';
  readonly category = 'ai';
  readonly description = 'Analyzes text sentiment';
  
  readonly configSchema = {
    type: 'object',
    properties: {
      threshold: {
        type: 'number',
        default: 0.1,
        minimum: 0,
        maximum: 1,
        description: 'Sentiment classification threshold',
      },
      languages: {
        type: 'array',
        items: { type: 'string' },
        default: ['en'],
        description: 'Supported languages',
      },
    },
  };

  readonly inputSchema = {
    type: 'object',
    properties: {
      text: {
        type: 'string',
        minLength: 1,
        description: 'Text to analyze',
      },
      lang: {
        type: 'string',
        default: 'en',
        description: 'Language code',
      },
    },
    required: ['text'],
  };

  readonly outputSchema = {
    type: 'object',
    properties: {
      sentiment: {
        type: 'string',
        enum: ['positive', 'negative', 'neutral'],
      },
      score: {
        type: 'number',
        minimum: -1,
        maximum: 1,
      },
      confidence: {
        type: 'number',
        minimum: 0,
        maximum: 1,
      },
      keywords: {
        type: 'array',
        items: { type: 'string' },
      },
    },
    required: ['sentiment', 'score', 'confidence', 'keywords'],
  };

  readonly examples = [
    {
      name: 'Positive sentiment',
      input: { text: 'This product is absolutely wonderful!' },
      output: {
        sentiment: 'positive' as const,
        score: 0.8,
        confidence: 0.9,
        keywords: ['wonderful'],
      },
    },
    {
      name: 'Negative sentiment',
      input: { text: 'Terrible service, very disappointed.' },
      output: {
        sentiment: 'negative' as const,
        score: -0.7,
        confidence: 0.85,
        keywords: ['terrible', 'disappointed'],
      },
    },
  ];

  // Word lists for simple sentiment analysis
  private readonly positiveWords = [
    'good', 'great', 'excellent', 'amazing', 'wonderful', 'fantastic',
    'love', 'like', 'best', 'happy', 'joy', 'brilliant', 'perfect',
    'beautiful', 'awesome', 'nice', 'super', 'fun', 'exciting',
  ];

  private readonly negativeWords = [
    'bad', 'terrible', 'awful', 'horrible', 'hate', 'dislike', 'worst',
    'sad', 'angry', 'disappointed', 'poor', 'fail', 'wrong', 'broken',
    'useless', 'waste', 'annoying', 'frustrating', 'boring',
  ];

  async prep(input: SentimentInput, config: SentimentConfig, store: any): Promise<any> {
    // Validate language support
    const supportedLangs = config.languages || ['en'];
    const lang = input.lang || 'en';
    
    if (!supportedLangs.includes(lang)) {
      throw new Error(`Language '${lang}' is not supported. Supported languages: ${supportedLangs.join(', ')}`);
    }

    // Clean and prepare text
    const cleanedText = input.text
      .toLowerCase()
      .replace(/[^\w\s]/g, ' ')
      .replace(/\s+/g, ' ')
      .trim();

    return {
      originalText: input.text,
      cleanedText,
      words: cleanedText.split(' ').filter(w => w.length > 0),
      lang,
    };
  }

  async exec(prepResult: any, config: SentimentConfig): Promise<SentimentOutput> {
    const { words, cleanedText } = prepResult;
    const threshold = config.threshold || 0.1;

    // Count positive and negative words
    let positiveCount = 0;
    let negativeCount = 0;
    const foundKeywords: string[] = [];

    for (const word of words) {
      if (this.positiveWords.includes(word)) {
        positiveCount++;
        foundKeywords.push(word);
      } else if (this.negativeWords.includes(word)) {
        negativeCount++;
        foundKeywords.push(word);
      }
    }

    // Calculate score
    const totalWords = words.length;
    const sentimentWords = positiveCount + negativeCount;
    
    let score = 0;
    if (sentimentWords > 0) {
      score = (positiveCount - negativeCount) / sentimentWords;
    }

    // Determine sentiment
    let sentiment: 'positive' | 'negative' | 'neutral';
    if (score > threshold) {
      sentiment = 'positive';
    } else if (score < -threshold) {
      sentiment = 'negative';
    } else {
      sentiment = 'neutral';
    }

    // Calculate confidence based on how many sentiment words were found
    const confidence = Math.min(sentimentWords / Math.max(totalWords * 0.1, 1), 1);

    return {
      sentiment,
      score: Math.round(score * 100) / 100,
      confidence: Math.round(confidence * 100) / 100,
      keywords: [...new Set(foundKeywords)], // Remove duplicates
    };
  }

  async post(
    input: SentimentInput,
    prepResult: any,
    execResult: SentimentOutput,
    config: SentimentConfig,
    store: any
  ): Promise<{ output: SentimentOutput; next: string }> {
    // Store analysis result for potential aggregation
    const timestamp = new Date().toISOString();
    store.set(`sentiment:${timestamp}`, {
      input: input.text,
      result: execResult,
      timestamp,
    });

    // Route based on sentiment
    let next = 'done';
    if (execResult.sentiment === 'positive' && execResult.confidence > 0.8) {
      next = 'high-positive';
    } else if (execResult.sentiment === 'negative' && execResult.confidence > 0.8) {
      next = 'high-negative';
    } else if (execResult.sentiment === 'neutral') {
      next = 'neutral';
    }

    return { output: execResult, next };
  }
}

// Create and initialize plugin
const plugin = new Plugin({
  name: 'sentiment-analyzer',
  version: '1.0.0',
  description: 'Sentiment analysis plugin for Pocket',
  author: 'Pocket Team',
  license: 'MIT',
  nodes: [], // Will be populated by register
  permissions: {
    memory: '10MB',
    timeout: 5000, // 5 seconds
  },
  requirements: {
    pocket: '>=1.0.0',
  },
});

// Register nodes
plugin.register(new SentimentAnalyzerNode());

// Initialize the plugin
initializePlugin(plugin);

// Export for bundling
export { plugin };