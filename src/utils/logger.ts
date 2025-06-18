/**
 * Secure logging utility for the New Relic Grafana plugin
 * Prevents sensitive information from being logged to the console
 */

export enum LogLevel {
  DEBUG = 'debug',
  INFO = 'info',
  WARN = 'warn',
  ERROR = 'error',
}

/**
 * Sanitizes log messages to remove sensitive information
 * @param message - The message to sanitize
 * @returns Sanitized message
 */
function sanitizeLogMessage(message: string): string {
  if (!message || typeof message !== 'string') {
    return '';
  }

  return message
    .replace(/apiKey['":\s]*['"]\w+['"]/gi, 'apiKey: "[REDACTED]"')
    .replace(/api[_-]?key['":\s]*['"]\w+['"]/gi, 'api_key: "[REDACTED]"')
    .replace(/password['":\s]*['"]\w+['"]/gi, 'password: "[REDACTED]"')
    .replace(/token['":\s]*['"]\w+['"]/gi, 'token: "[REDACTED]"')
    .replace(/secret['":\s]*['"]\w+['"]/gi, 'secret: "[REDACTED]"');
}

/**
 * Sanitizes objects to remove sensitive fields
 * @param obj - The object to sanitize
 * @returns Sanitized object
 */
function sanitizeObject(obj: any): any {
  if (!obj || typeof obj !== 'object') {
    return obj;
  }

  const sensitiveFields = ['apiKey', 'api_key', 'password', 'token', 'secret'];
  const sanitized = { ...obj };

  for (const field of sensitiveFields) {
    if (field in sanitized) {
      sanitized[field] = '[REDACTED]';
    }
  }

  return sanitized;
}

/**
 * Secure logger class
 */
class SecureLogger {
  private isDevelopment: boolean;

  constructor() {
    this.isDevelopment = process.env.NODE_ENV === 'development';
  }

  /**
   * Logs a debug message (only in development)
   * @param message - The message to log
   * @param data - Optional data to log
   */
  debug(message: string, data?: any): void {
    if (this.isDevelopment) {
      const sanitizedMessage = sanitizeLogMessage(message);
      const sanitizedData = data ? sanitizeObject(data) : undefined;
      
      if (sanitizedData) {
        console.debug(`[NewRelic Plugin] ${sanitizedMessage}`, sanitizedData);
      } else {
        console.debug(`[NewRelic Plugin] ${sanitizedMessage}`);
      }
    }
  }

  /**
   * Logs an info message
   * @param message - The message to log
   * @param data - Optional data to log
   */
  info(message: string, data?: any): void {
    const sanitizedMessage = sanitizeLogMessage(message);
    const sanitizedData = data ? sanitizeObject(data) : undefined;
    
    if (sanitizedData) {
      console.info(`[NewRelic Plugin] ${sanitizedMessage}`, sanitizedData);
    } else {
      console.info(`[NewRelic Plugin] ${sanitizedMessage}`);
    }
  }

  /**
   * Logs a warning message
   * @param message - The message to log
   * @param data - Optional data to log
   */
  warn(message: string, data?: any): void {
    const sanitizedMessage = sanitizeLogMessage(message);
    const sanitizedData = data ? sanitizeObject(data) : undefined;
    
    if (sanitizedData) {
      console.warn(`[NewRelic Plugin] ${sanitizedMessage}`, sanitizedData);
    } else {
      console.warn(`[NewRelic Plugin] ${sanitizedMessage}`);
    }
  }

  /**
   * Logs an error message
   * @param message - The message to log
   * @param error - Optional error object
   * @param data - Optional additional data
   */
  error(message: string, error?: Error, data?: any): void {
    const sanitizedMessage = sanitizeLogMessage(message);
    const sanitizedData = data ? sanitizeObject(data) : undefined;
    
    if (error && sanitizedData) {
      console.error(`[NewRelic Plugin] ${sanitizedMessage}`, error, sanitizedData);
    } else if (error) {
      console.error(`[NewRelic Plugin] ${sanitizedMessage}`, error);
    } else if (sanitizedData) {
      console.error(`[NewRelic Plugin] ${sanitizedMessage}`, sanitizedData);
    } else {
      console.error(`[NewRelic Plugin] ${sanitizedMessage}`);
    }
  }
}

// Export singleton instance
export const logger = new SecureLogger();

// Export individual log functions for convenience
export const { debug, info, warn, error } = logger; 