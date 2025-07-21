#!/usr/bin/env node

const esbuild = require('esbuild');
const fs = require('fs');
const path = require('path');

// Javy wrapper that handles the plugin lifecycle
const javyWrapper = `
// Import the compiled plugin
import { plugin } from './index.js';
import { initializePlugin } from '@pocket/plugin-sdk';

// Initialize the plugin
initializePlugin(plugin);

// Javy IO handling
const Javy = globalThis.Javy;

// Main entry point for Javy
if (typeof Javy !== 'undefined' && Javy.IO) {
  try {
    // Read input from stdin
    const inputBytes = Javy.IO.readSync(0);
    const inputStr = new TextDecoder().decode(inputBytes);
    
    let output;
    if (!inputStr || inputStr.trim() === '') {
      // Return metadata
      output = {
        success: true,
        output: plugin.metadata
      };
    } else {
      // Parse and handle request
      const request = JSON.parse(inputStr);
      
      // Since Javy doesn't support async well, we need to handle this carefully
      // For now, we'll note that full async support requires runtime changes
      output = {
        success: false,
        error: 'Async plugin operations require host runtime support'
      };
    }
    
    // Write output
    const outputStr = JSON.stringify(output);
    Javy.IO.writeSync(1, new TextEncoder().encode(outputStr));
  } catch (error) {
    const errorOutput = JSON.stringify({
      success: false,
      error: error.message || String(error)
    });
    Javy.IO.writeSync(1, new TextEncoder().encode(errorOutput));
  }
} else {
  // Non-Javy environment (for testing)
  console.log('Plugin loaded:', plugin.metadata.name);
}
`;

async function build() {
  console.log('Building TypeScript...');
  
  // First compile TypeScript
  const { execSync } = require('child_process');
  execSync('npx tsc', { stdio: 'inherit' });
  
  console.log('Bundling for Javy...');
  
  // Create the wrapper file
  const wrapperPath = path.join('dist', 'javy-wrapper.js');
  fs.writeFileSync(wrapperPath, javyWrapper);
  
  // Bundle with esbuild
  await esbuild.build({
    entryPoints: [wrapperPath],
    bundle: true,
    platform: 'neutral',
    target: 'es2020',
    format: 'iife',
    outfile: 'dist/plugin.js',
    // Important settings for Javy
    define: {
      'process.env.NODE_ENV': '"production"',
      'global': 'globalThis'
    },
    // External modules that Javy provides
    external: ['javy'],
    // Don't minify - Javy handles optimization
    minify: false,
    // No source maps for WASM
    sourcemap: false,
    // Ensure all code is included
    treeShaking: false,
  });
  
  console.log('✓ Bundle created at dist/plugin.js');
  
  // Check if Javy is installed
  try {
    execSync('javy --version', { stdio: 'ignore' });
    console.log('✓ Javy is installed');
    console.log('\nTo compile to WASM, run:');
    console.log('  npm run build:wasm');
    console.log('  # or');
    console.log('  javy compile dist/plugin.js -o plugin.wasm');
  } catch (e) {
    console.log('\n⚠️  Javy is not installed');
    console.log('Install with: npm install -g @shopify/javy');
  }
}

// Support watch mode
if (process.argv.includes('--watch')) {
  console.log('Watch mode not implemented yet. Run build manually.');
} else {
  build().catch(err => {
    console.error('Build failed:', err);
    process.exit(1);
  });
}