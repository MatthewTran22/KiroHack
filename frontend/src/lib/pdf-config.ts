import { pdfjs } from 'react-pdf';

// Configure PDF.js worker for Next.js environment
const configurePDFWorker = () => {
  if (typeof window !== 'undefined' && !pdfjs.GlobalWorkerOptions.workerSrc) {
    // Use the local worker file from public directory
    pdfjs.GlobalWorkerOptions.workerSrc = '/pdf.worker.min.js';
  }
};

// Initialize worker configuration
configurePDFWorker();

export { pdfjs, configurePDFWorker };
