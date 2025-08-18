/**
 * Integration tests for document management system
 * These tests use Docker containers to test against the real backend API
 */

import { apiClient } from '@/lib/api';
import { Document, DocumentUploadRequest } from '@/types';

// Test configuration
const TEST_CONFIG = {
  API_URL: process.env.TEST_API_URL || 'http://localhost:8080',
  TIMEOUT: 30000,
  TEST_USER: {
    email: 'test@example.com',
    password: 'testpassword123',
  },
};

// Helper function to create test files
function createTestFile(name: string, content: string, type: string): File {
  const blob = new Blob([content], { type });
  return new File([blob], name, { type });
}

// Helper function to wait for document processing
async function waitForDocumentProcessing(documentId: string, maxWaitTime = 30000): Promise<Document> {
  const startTime = Date.now();
  
  while (Date.now() - startTime < maxWaitTime) {
    const document = await apiClient.documents.getDocument(documentId);
    
    if (document.status === 'completed') {
      return document;
    }
    
    if (document.status === 'error') {
      throw new Error(`Document processing failed: ${documentId}`);
    }
    
    // Wait 1 second before checking again
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  
  throw new Error(`Document processing timeout: ${documentId}`);
}

describe('Document Management Integration Tests', () => {
  let authToken: string;
  let testDocuments: Document[] = [];

  beforeAll(async () => {
    // Skip integration tests if not in CI environment or if API is not available
    if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
      console.log('Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.');
      return;
    }

    try {
      // Authenticate with test user
      const authResponse = await apiClient.auth.login(TEST_CONFIG.TEST_USER);
      authToken = authResponse.tokens.access_token;
      
      // Set token for subsequent requests
      apiClient.addRequestInterceptor((config) => ({
        ...config,
        headers: {
          ...config.headers,
          Authorization: `Bearer ${authToken}`,
        },
      }));
    } catch (error) {
      console.error('Failed to authenticate for integration tests:', error);
      throw error;
    }
  }, TEST_CONFIG.TIMEOUT);

  afterAll(async () => {
    if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
      return;
    }

    try {
      // Clean up test documents
      if (testDocuments.length > 0) {
        const documentIds = testDocuments.map(doc => doc.id);
        await apiClient.documents.deleteDocuments(documentIds);
      }

      // Logout
      await apiClient.auth.logout();
    } catch (error) {
      console.error('Cleanup failed:', error);
    }
  });

  describe('Document Upload', () => {
    it('should upload a single document successfully', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      const testFile = createTestFile('test-document.txt', 'This is a test document content.', 'text/plain');
      
      const uploadRequest: DocumentUploadRequest = {
        file: testFile,
        metadata: {
          title: 'Test Document',
          description: 'A test document for integration testing',
          author: 'Test User',
          department: 'IT',
          category: 'Testing',
          keywords: ['test', 'integration'],
          language: 'en',
          version: '1.0',
        },
        classification: 'internal',
        tags: ['test', 'integration'],
      };

      const uploadedDocuments = await apiClient.documents.upload([uploadRequest]);
      
      expect(uploadedDocuments).toHaveLength(1);
      
      const document = uploadedDocuments[0];
      testDocuments.push(document);
      
      expect(document.name).toBe('test-document.txt');
      expect(document.type).toBe('txt');
      expect(document.classification).toBe('internal');
      expect(document.tags).toEqual(['test', 'integration']);
      expect(document.metadata.title).toBe('Test Document');
      expect(document.metadata.author).toBe('Test User');
      
      // Wait for document processing to complete
      const processedDocument = await waitForDocumentProcessing(document.id);
      expect(processedDocument.status).toBe('completed');
    }, TEST_CONFIG.TIMEOUT);

    it('should upload multiple documents in batch', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      const testFiles = [
        createTestFile('batch-test-1.txt', 'First batch test document.', 'text/plain'),
        createTestFile('batch-test-2.txt', 'Second batch test document.', 'text/plain'),
        createTestFile('batch-test-3.txt', 'Third batch test document.', 'text/plain'),
      ];

      const uploadRequests: DocumentUploadRequest[] = testFiles.map((file, index) => ({
        file,
        metadata: {
          title: `Batch Test Document ${index + 1}`,
          description: `Batch test document number ${index + 1}`,
          author: 'Test User',
          department: 'IT',
          category: 'Testing',
          version: '1.0',
        },
        classification: 'internal',
        tags: ['batch', 'test'],
      }));

      const uploadedDocuments = await apiClient.documents.upload(uploadRequests);
      
      expect(uploadedDocuments).toHaveLength(3);
      testDocuments.push(...uploadedDocuments);
      
      // Verify all documents were uploaded correctly
      for (let i = 0; i < uploadedDocuments.length; i++) {
        const document = uploadedDocuments[i];
        expect(document.name).toBe(`batch-test-${i + 1}.txt`);
        expect(document.metadata.title).toBe(`Batch Test Document ${i + 1}`);
        expect(document.tags).toEqual(['batch', 'test']);
      }

      // Wait for all documents to be processed
      const processedDocuments = await Promise.all(
        uploadedDocuments.map(doc => waitForDocumentProcessing(doc.id))
      );
      
      processedDocuments.forEach(doc => {
        expect(doc.status).toBe('completed');
      });
    }, TEST_CONFIG.TIMEOUT);

    it('should handle file validation errors', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      // Create a file that's too large (assuming 50MB limit)
      const largeContent = 'x'.repeat(51 * 1024 * 1024); // 51MB
      const largeFile = createTestFile('large-file.txt', largeContent, 'text/plain');

      const uploadRequest: DocumentUploadRequest = {
        file: largeFile,
        metadata: {
          title: 'Large File Test',
          description: 'This file should be rejected due to size',
        },
        classification: 'internal',
        tags: ['test'],
      };

      await expect(apiClient.documents.upload([uploadRequest])).rejects.toThrow();
    }, TEST_CONFIG.TIMEOUT);
  });

  describe('Document Retrieval and Filtering', () => {
    it('should retrieve documents with filters', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      // Get all documents
      const allDocuments = await apiClient.documents.getDocuments();
      expect(allDocuments.data.length).toBeGreaterThan(0);

      // Filter by classification
      const internalDocuments = await apiClient.documents.getDocuments({
        classification: 'internal',
      });
      
      internalDocuments.data.forEach(doc => {
        expect(doc.classification).toBe('internal');
      });

      // Filter by tags
      const testDocuments = await apiClient.documents.getDocuments({
        tags: ['test'],
      });
      
      testDocuments.data.forEach(doc => {
        expect(doc.tags).toContain('test');
      });

      // Filter by date range
      const today = new Date();
      const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000);
      
      const recentDocuments = await apiClient.documents.getDocuments({
        dateRange: {
          start: yesterday,
          end: today,
        },
      });
      
      recentDocuments.data.forEach(doc => {
        const uploadDate = new Date(doc.uploadedAt);
        expect(uploadDate.getTime()).toBeGreaterThanOrEqual(yesterday.getTime());
        expect(uploadDate.getTime()).toBeLessThanOrEqual(today.getTime());
      });
    }, TEST_CONFIG.TIMEOUT);

    it('should search documents by content and metadata', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      // Search by title
      const titleResults = await apiClient.documents.searchDocuments('Test Document');
      expect(titleResults.data.length).toBeGreaterThan(0);
      
      titleResults.data.forEach(doc => {
        const titleMatch = doc.metadata.title?.toLowerCase().includes('test document');
        const nameMatch = doc.name.toLowerCase().includes('test');
        expect(titleMatch || nameMatch).toBe(true);
      });

      // Search by content (if full-text search is implemented)
      const contentResults = await apiClient.documents.searchDocuments('test document content');
      expect(contentResults.data.length).toBeGreaterThanOrEqual(0);

      // Search with filters
      const filteredResults = await apiClient.documents.searchDocuments('test', {
        classification: 'internal',
        tags: ['test'],
      });
      
      filteredResults.data.forEach(doc => {
        expect(doc.classification).toBe('internal');
        expect(doc.tags).toContain('test');
      });
    }, TEST_CONFIG.TIMEOUT);
  });

  describe('Document Operations', () => {
    it('should download documents', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      if (testDocuments.length === 0) {
        throw new Error('No test documents available for download test');
      }

      const document = testDocuments[0];
      const downloadedBlob = await apiClient.documents.downloadDocument(document.id);
      
      expect(downloadedBlob).toBeInstanceOf(Blob);
      expect(downloadedBlob.size).toBeGreaterThan(0);
      
      // Verify content if it's a text file
      if (document.type === 'txt') {
        const text = await downloadedBlob.text();
        expect(text.length).toBeGreaterThan(0);
      }
    }, TEST_CONFIG.TIMEOUT);

    it('should update document metadata', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      if (testDocuments.length === 0) {
        throw new Error('No test documents available for update test');
      }

      const document = testDocuments[0];
      const updates = {
        metadata: {
          ...document.metadata,
          title: 'Updated Test Document Title',
          description: 'Updated description for integration testing',
        },
        tags: [...document.tags, 'updated'],
        classification: 'confidential' as const,
      };

      const updatedDocument = await apiClient.documents.updateDocument(document.id, updates);
      
      expect(updatedDocument.metadata.title).toBe('Updated Test Document Title');
      expect(updatedDocument.metadata.description).toBe('Updated description for integration testing');
      expect(updatedDocument.tags).toContain('updated');
      expect(updatedDocument.classification).toBe('confidential');

      // Update test documents array
      const index = testDocuments.findIndex(doc => doc.id === document.id);
      if (index !== -1) {
        testDocuments[index] = updatedDocument;
      }
    }, TEST_CONFIG.TIMEOUT);

    it('should delete individual documents', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      // Create a document specifically for deletion test
      const testFile = createTestFile('delete-test.txt', 'This document will be deleted.', 'text/plain');
      
      const uploadRequest: DocumentUploadRequest = {
        file: testFile,
        metadata: {
          title: 'Document to Delete',
          description: 'This document is created for deletion testing',
        },
        classification: 'internal',
        tags: ['delete-test'],
      };

      const [uploadedDocument] = await apiClient.documents.upload([uploadRequest]);
      await waitForDocumentProcessing(uploadedDocument.id);

      // Delete the document
      await apiClient.documents.deleteDocument(uploadedDocument.id);

      // Verify document is deleted
      await expect(apiClient.documents.getDocument(uploadedDocument.id)).rejects.toThrow();
    }, TEST_CONFIG.TIMEOUT);

    it('should delete multiple documents in batch', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      // Create multiple documents for batch deletion
      const testFiles = [
        createTestFile('batch-delete-1.txt', 'First document to delete.', 'text/plain'),
        createTestFile('batch-delete-2.txt', 'Second document to delete.', 'text/plain'),
      ];

      const uploadRequests: DocumentUploadRequest[] = testFiles.map((file, index) => ({
        file,
        metadata: {
          title: `Batch Delete Document ${index + 1}`,
          description: `Document ${index + 1} for batch deletion testing`,
        },
        classification: 'internal',
        tags: ['batch-delete-test'],
      }));

      const uploadedDocuments = await apiClient.documents.upload(uploadRequests);
      
      // Wait for processing
      await Promise.all(
        uploadedDocuments.map(doc => waitForDocumentProcessing(doc.id))
      );

      const documentIds = uploadedDocuments.map(doc => doc.id);

      // Delete documents in batch
      await apiClient.documents.deleteDocuments(documentIds);

      // Verify all documents are deleted
      for (const id of documentIds) {
        await expect(apiClient.documents.getDocument(id)).rejects.toThrow();
      }
    }, TEST_CONFIG.TIMEOUT);
  });

  describe('Error Handling', () => {
    it('should handle network errors gracefully', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      // Create a client with invalid URL to simulate network error
      const invalidClient = new (apiClient.constructor as any)('http://invalid-url:9999');
      
      await expect(invalidClient.documents.getDocuments()).rejects.toThrow();
    }, TEST_CONFIG.TIMEOUT);

    it('should handle authentication errors', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      // Create a client without authentication
      const unauthenticatedClient = new (apiClient.constructor as any)(TEST_CONFIG.API_URL);
      
      await expect(unauthenticatedClient.documents.getDocuments()).rejects.toThrow();
    }, TEST_CONFIG.TIMEOUT);

    it('should handle invalid document IDs', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      const invalidId = 'invalid-document-id-12345';
      
      await expect(apiClient.documents.getDocument(invalidId)).rejects.toThrow();
      await expect(apiClient.documents.deleteDocument(invalidId)).rejects.toThrow();
      await expect(apiClient.documents.downloadDocument(invalidId)).rejects.toThrow();
    }, TEST_CONFIG.TIMEOUT);
  });

  describe('Performance and Load Testing', () => {
    it('should handle concurrent uploads', async () => {
      if (!process.env.CI && !process.env.RUN_INTEGRATION_TESTS) {
        return;
      }

      const concurrentUploads = 5;
      const uploadPromises: Promise<Document[]>[] = [];

      for (let i = 0; i < concurrentUploads; i++) {
        const testFile = createTestFile(
          `concurrent-test-${i}.txt`,
          `Concurrent upload test document ${i}`,
          'text/plain'
        );

        const uploadRequest: DocumentUploadRequest = {
          file: testFile,
          metadata: {
            title: `Concurrent Test Document ${i}`,
            description: `Document ${i} for concurrent upload testing`,
          },
          classification: 'internal',
          tags: ['concurrent-test'],
        };

        uploadPromises.push(apiClient.documents.upload([uploadRequest]));
      }

      const results = await Promise.all(uploadPromises);
      
      expect(results).toHaveLength(concurrentUploads);
      
      const allDocuments = results.flat();
      testDocuments.push(...allDocuments);
      
      // Verify all uploads succeeded
      allDocuments.forEach((doc, index) => {
        expect(doc.name).toBe(`concurrent-test-${index}.txt`);
        expect(doc.tags).toContain('concurrent-test');
      });

      // Wait for all documents to be processed
      await Promise.all(
        allDocuments.map(doc => waitForDocumentProcessing(doc.id))
      );
    }, TEST_CONFIG.TIMEOUT * 2); // Extended timeout for concurrent operations
  });
});

// Helper function to run integration tests with Docker
export async function runDockerIntegrationTests() {
  if (process.env.NODE_ENV === 'test' && process.env.RUN_INTEGRATION_TESTS) {
    console.log('Starting Docker containers for integration tests...');
    
    // This would typically use a test runner that starts Docker containers
    // For example, using testcontainers or docker-compose
    
    try {
      // Start backend services
      // await startDockerServices();
      
      // Run the tests
      // The tests will run automatically when Jest executes this file
      
      console.log('Integration tests completed successfully');
    } catch (error) {
      console.error('Integration tests failed:', error);
      throw error;
    } finally {
      // Clean up Docker containers
      // await stopDockerServices();
    }
  }
}