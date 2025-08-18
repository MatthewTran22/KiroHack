import { apiClient, APIError } from '../../lib/api';
import { tokenManager } from '../../lib/auth';

// Integration tests that run against the actual backend in Docker containers
describe('API Integration Tests', () => {
  const testUser = {
    email: 'test@example.com',
    password: 'testpassword123',
    name: 'Test User',
  };

  let authToken: string;
  let refreshToken: string;
  let userId: string;

  beforeAll(async () => {
    // Wait for backend to be ready
    await waitForBackend();
  });

  afterAll(async () => {
    // Cleanup test data
    if (authToken) {
      tokenManager.setTokens(authToken, refreshToken);
      try {
        await apiClient.auth.logout();
      } catch (error) {
        console.warn('Cleanup logout failed:', error);
      }
    }
  });

  describe('Authentication Flow', () => {
    it('should complete full authentication flow', async () => {
      // Test login
      const loginResponse = await apiClient.auth.login({
        email: testUser.email,
        password: testUser.password,
      });

      expect(loginResponse.user).toBeDefined();
      expect(loginResponse.token).toBeDefined();
      expect(loginResponse.refreshToken).toBeDefined();
      expect(loginResponse.user.email).toBe(testUser.email);

      authToken = loginResponse.token;
      refreshToken = loginResponse.refreshToken;
      userId = loginResponse.user.id;

      // Set tokens for subsequent requests
      tokenManager.setTokens(authToken, refreshToken);

      // Test getting current user
      const currentUser = await apiClient.auth.getCurrentUser();
      expect(currentUser.id).toBe(userId);
      expect(currentUser.email).toBe(testUser.email);

      // Test token refresh
      const refreshResponse = await apiClient.auth.refreshToken();
      expect(refreshResponse.token).toBeDefined();
      expect(refreshResponse.refreshToken).toBeDefined();
      expect(refreshResponse.user.id).toBe(userId);

      // Update tokens
      authToken = refreshResponse.token;
      refreshToken = refreshResponse.refreshToken;
      tokenManager.setTokens(authToken, refreshToken);
    });

    it('should handle invalid credentials', async () => {
      await expect(
        apiClient.auth.login({
          email: testUser.email,
          password: 'wrongpassword',
        })
      ).rejects.toThrow(APIError);
    });

    it('should handle missing refresh token', async () => {
      tokenManager.clearTokens();
      
      await expect(apiClient.auth.refreshToken()).rejects.toThrow(APIError);
    });
  });

  describe('Documents API', () => {
    let documentId: string;

    beforeEach(() => {
      tokenManager.setTokens(authToken, refreshToken);
    });

    it('should upload and manage documents', async () => {
      // Create test file
      const testContent = 'This is a test document for integration testing.';
      const testFile = new File([testContent], 'test-document.txt', {
        type: 'text/plain',
      });

      // Test document upload
      const uploadResponse = await apiClient.documents.upload([
        {
          file: testFile,
          metadata: {
            title: 'Integration Test Document',
            description: 'A document created during integration testing',
            category: 'test',
          },
          tags: ['integration', 'test'],
          classification: 'internal',
        },
      ]);

      expect(uploadResponse).toHaveLength(1);
      expect(uploadResponse[0].name).toBe('test-document.txt');
      expect(uploadResponse[0].metadata.title).toBe('Integration Test Document');
      expect(uploadResponse[0].tags).toContain('integration');

      documentId = uploadResponse[0].id;

      // Test getting documents
      const documentsResponse = await apiClient.documents.getDocuments({
        tags: ['integration'],
      });

      expect(documentsResponse.data).toHaveLength(1);
      expect(documentsResponse.data[0].id).toBe(documentId);

      // Test getting single document
      const document = await apiClient.documents.getDocument(documentId);
      expect(document.id).toBe(documentId);
      expect(document.name).toBe('test-document.txt');

      // Test updating document
      const updatedDocument = await apiClient.documents.updateDocument(documentId, {
        metadata: {
          ...document.metadata,
          description: 'Updated description',
        },
      });

      expect(updatedDocument.metadata.description).toBe('Updated description');

      // Test searching documents
      const searchResponse = await apiClient.documents.searchDocuments('integration');
      expect(searchResponse.data.length).toBeGreaterThan(0);
      expect(searchResponse.data.some(doc => doc.id === documentId)).toBe(true);
    });

    it('should handle document download', async () => {
      if (!documentId) {
        throw new Error('Document ID not available for download test');
      }

      const blob = await apiClient.documents.downloadDocument(documentId);
      expect(blob).toBeInstanceOf(Blob);
      expect(blob.size).toBeGreaterThan(0);

      // Verify content
      const text = await blob.text();
      expect(text).toContain('This is a test document');
    });

    it('should handle document deletion', async () => {
      if (!documentId) {
        throw new Error('Document ID not available for deletion test');
      }

      await apiClient.documents.deleteDocument(documentId);

      // Verify document is deleted
      await expect(
        apiClient.documents.getDocument(documentId)
      ).rejects.toThrow(APIError);
    });

    it('should handle file validation errors', async () => {
      // Test with oversized file (if backend has size limits)
      const largeContent = 'x'.repeat(100 * 1024 * 1024); // 100MB
      const largeFile = new File([largeContent], 'large-file.txt', {
        type: 'text/plain',
      });

      await expect(
        apiClient.documents.upload([
          {
            file: largeFile,
            metadata: { title: 'Large File' },
          },
        ])
      ).rejects.toThrow(APIError);
    });
  });

  describe('Consultations API', () => {
    let consultationId: string;

    beforeEach(() => {
      tokenManager.setTokens(authToken, refreshToken);
    });

    it('should create and manage consultations', async () => {
      // Test creating consultation
      const consultation = await apiClient.consultations.createSession({
        type: 'policy',
        title: 'Integration Test Consultation',
        priority: 'medium',
        context: 'This is a test consultation for integration testing',
      });

      expect(consultation.title).toBe('Integration Test Consultation');
      expect(consultation.type).toBe('policy');
      expect(consultation.priority).toBe('medium');
      expect(consultation.status).toBe('active');

      consultationId = consultation.id;

      // Test getting consultations
      const consultationsResponse = await apiClient.consultations.getSessions({
        type: 'policy',
        status: 'active',
      });

      expect(consultationsResponse.data.length).toBeGreaterThan(0);
      expect(consultationsResponse.data.some(c => c.id === consultationId)).toBe(true);

      // Test getting single consultation
      const retrievedConsultation = await apiClient.consultations.getSession(consultationId);
      expect(retrievedConsultation.id).toBe(consultationId);
      expect(retrievedConsultation.title).toBe('Integration Test Consultation');
    });

    it('should handle consultation messages', async () => {
      if (!consultationId) {
        throw new Error('Consultation ID not available for message test');
      }

      // Test sending message
      const message = await apiClient.consultations.sendMessage(consultationId, {
        content: 'What are the key considerations for policy implementation?',
        inputMethod: 'text',
      });

      expect(message.content).toBe('What are the key considerations for policy implementation?');
      expect(message.type).toBe('user');
      expect(message.sessionId).toBe(consultationId);
      expect(message.inputMethod).toBe('text');

      // Test getting messages
      const messages = await apiClient.consultations.getMessages(consultationId);
      expect(messages.length).toBeGreaterThan(0);
      expect(messages.some(m => m.id === message.id)).toBe(true);
    });

    it('should handle consultation search', async () => {
      const searchResponse = await apiClient.consultations.searchSessions('integration');
      expect(searchResponse.data.length).toBeGreaterThan(0);
      
      if (consultationId) {
        expect(searchResponse.data.some(c => c.id === consultationId)).toBe(true);
      }
    });

    it('should handle consultation updates', async () => {
      if (!consultationId) {
        throw new Error('Consultation ID not available for update test');
      }

      const updatedConsultation = await apiClient.consultations.updateSession(consultationId, {
        status: 'completed',
        summary: 'Integration test completed successfully',
      });

      expect(updatedConsultation.status).toBe('completed');
      expect(updatedConsultation.summary).toBe('Integration test completed successfully');
    });

    it('should handle consultation export', async () => {
      if (!consultationId) {
        throw new Error('Consultation ID not available for export test');
      }

      const exportBlob = await apiClient.consultations.exportSession(consultationId, 'json');
      expect(exportBlob).toBeInstanceOf(Blob);
      expect(exportBlob.size).toBeGreaterThan(0);

      // Verify export content
      const exportText = await exportBlob.text();
      const exportData = JSON.parse(exportText);
      expect(exportData.id).toBe(consultationId);
      expect(exportData.title).toBe('Integration Test Consultation');
    });

    afterAll(async () => {
      // Cleanup consultation
      if (consultationId) {
        try {
          await apiClient.consultations.deleteSession(consultationId);
        } catch (error) {
          console.warn('Cleanup consultation deletion failed:', error);
        }
      }
    });
  });

  describe('Error Handling', () => {
    beforeEach(() => {
      tokenManager.setTokens(authToken, refreshToken);
    });

    it('should handle network timeouts', async () => {
      // Create a client with very short timeout
      const timeoutClient = new (apiClient.constructor as any)();
      timeoutClient.defaultRetryConfig.maxAttempts = 1;

      // This should timeout quickly
      await expect(
        timeoutClient.request('/api/v1/slow-endpoint', { timeout: 1 })
      ).rejects.toThrow(APIError);
    }, 10000);

    it('should handle server errors with retry', async () => {
      // Test endpoint that might return server errors
      try {
        await apiClient.consultations.createSession({
          type: 'invalid-type' as any,
          title: 'Invalid Consultation',
        });
      } catch (error) {
        expect(error).toBeInstanceOf(APIError);
        expect((error as APIError).status).toBeGreaterThanOrEqual(400);
      }
    });

    it('should handle unauthorized access', async () => {
      // Clear tokens to simulate unauthorized access
      tokenManager.clearTokens();

      await expect(
        apiClient.consultations.getSessions()
      ).rejects.toThrow(APIError);

      // Restore tokens
      tokenManager.setTokens(authToken, refreshToken);
    });
  });

  describe('Audit API', () => {
    beforeEach(() => {
      tokenManager.setTokens(authToken, refreshToken);
    });

    it('should retrieve audit logs', async () => {
      // Perform some actions to generate audit logs
      await apiClient.auth.getCurrentUser();
      
      // Wait a bit for audit logs to be written
      await new Promise(resolve => setTimeout(resolve, 1000));

      const auditResponse = await apiClient.audit.getLogs({
        userId: userId,
        action: 'get_current_user',
      });

      expect(auditResponse.data).toBeDefined();
      expect(Array.isArray(auditResponse.data)).toBe(true);
    });

    it('should export audit logs', async () => {
      const exportBlob = await apiClient.audit.exportLogs({
        userId: userId,
      }, 'csv');

      expect(exportBlob).toBeInstanceOf(Blob);
      expect(exportBlob.type).toContain('csv');
    });
  });
});

// Helper function to wait for backend to be ready
async function waitForBackend(maxAttempts = 30, delay = 1000): Promise<void> {
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      const response = await fetch('http://localhost:8080/health');
      if (response.ok) {
        return;
      }
    } catch (error) {
      // Backend not ready yet
    }

    if (attempt === maxAttempts) {
      throw new Error('Backend did not become ready within the expected time');
    }

    await new Promise(resolve => setTimeout(resolve, delay));
  }
}