// Polyfill for Node 20 compatibility
if (typeof globalThis !== 'undefined' && !require('util').styleText) {
  const util = require('util');
  util.styleText = (format, text) => {
    // Simple implementation - just return the text
    // In a real scenario, you'd apply ANSI codes based on format
    return text;
  };
}
