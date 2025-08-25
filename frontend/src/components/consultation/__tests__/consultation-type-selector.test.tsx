import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConsultationTypeSelector } from '../consultation-type-selector';

const mockOnStartConsultation = jest.fn();

const defaultProps = {
    onStartConsultation: mockOnStartConsultation,
    isLoading: false,
};

describe('ConsultationTypeSelector', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('renders consultation type options', () => {
        render(<ConsultationTypeSelector {...defaultProps} />);

        expect(screen.getByText('Choose Consultation Type')).toBeInTheDocument();
        expect(screen.getByText('Policy Analysis')).toBeInTheDocument();
        expect(screen.getByText('Strategic Planning')).toBeInTheDocument();
        expect(screen.getByText('Operations Management')).toBeInTheDocument();
        expect(screen.getByText('Technology Advisory')).toBeInTheDocument();
        expect(screen.getByText('Research & Analysis')).toBeInTheDocument();
        expect(screen.getByText('Compliance & Risk')).toBeInTheDocument();
    });

    it('shows type descriptions and examples', () => {
        render(<ConsultationTypeSelector {...defaultProps} />);

        expect(screen.getByText(/Analyze existing policies, draft new regulations/)).toBeInTheDocument();
        expect(screen.getByText('Policy impact assessment')).toBeInTheDocument();
        expect(screen.getByText('Strategic planning')).toBeInTheDocument();
    });

    it('handles quick start for consultation types', async () => {
        const user = userEvent.setup();
        render(<ConsultationTypeSelector {...defaultProps} />);

        const quickStartButtons = screen.getAllByText('Quick Start');
        await user.click(quickStartButtons[0]); // Click first quick start button

        expect(mockOnStartConsultation).toHaveBeenCalledWith('policy');
    });

    it('shows detailed form when type is selected', async () => {
        const user = userEvent.setup();
        render(<ConsultationTypeSelector {...defaultProps} />);

        const policyCard = screen.getByText('Policy Analysis').closest('[role="button"]') ||
            screen.getByText('Policy Analysis').closest('div');

        if (policyCard) {
            await user.click(policyCard);
        }

        await waitFor(() => {
            expect(screen.getByText('Consultation Details')).toBeInTheDocument();
            expect(screen.getByLabelText('Session Title')).toBeInTheDocument();
            expect(screen.getByLabelText(/Context & Background/)).toBeInTheDocument();
        });
    });

    it('handles form submission with title and context', async () => {
        const user = userEvent.setup();
        render(<ConsultationTypeSelector {...defaultProps} />);

        // Select policy type
        const policyCard = screen.getByText('Policy Analysis').closest('div');
        if (policyCard) {
            await user.click(policyCard);
        }

        await waitFor(() => {
            expect(screen.getByLabelText('Session Title')).toBeInTheDocument();
        });

        // Fill form
        const titleInput = screen.getByLabelText('Session Title');
        const contextTextarea = screen.getByLabelText(/Context & Background/);

        await user.clear(titleInput);
        await user.type(titleInput, 'Test Policy Session');
        await user.type(contextTextarea, 'Test context for policy analysis');

        // Submit
        const startButton = screen.getByText('Start Consultation');
        await user.click(startButton);

        expect(mockOnStartConsultation).toHaveBeenCalledWith(
            'policy',
            'Test Policy Session',
            'Test context for policy analysis'
        );
    });

    it('disables start button when title is empty', async () => {
        const user = userEvent.setup();
        render(<ConsultationTypeSelector {...defaultProps} />);

        // Select policy type
        const policyCard = screen.getByText('Policy Analysis').closest('div');
        if (policyCard) {
            await user.click(policyCard);
        }

        await waitFor(() => {
            expect(screen.getByLabelText('Session Title')).toBeInTheDocument();
        });

        // Clear title
        const titleInput = screen.getByLabelText('Session Title');
        await user.clear(titleInput);

        const startButton = screen.getByText('Start Consultation');
        expect(startButton).toBeDisabled();
    });

    it('shows loading state', () => {
        render(<ConsultationTypeSelector {...defaultProps} isLoading={true} />);

        const quickStartButtons = screen.getAllByText('Quick Start');
        quickStartButtons.forEach(button => {
            expect(button).toBeDisabled();
        });
    });

    it('allows navigation back from details form', async () => {
        const user = userEvent.setup();
        render(<ConsultationTypeSelector {...defaultProps} />);

        // Select type
        const policyCard = screen.getByText('Policy Analysis').closest('div');
        if (policyCard) {
            await user.click(policyCard);
        }

        await waitFor(() => {
            expect(screen.getByText('Consultation Details')).toBeInTheDocument();
        });

        // Go back
        const backButton = screen.getByText('← Back');
        await user.click(backButton);

        await waitFor(() => {
            expect(screen.getByText('Choose Consultation Type')).toBeInTheDocument();
        });
    });

    it('shows example use cases for selected type', async () => {
        const user = userEvent.setup();
        render(<ConsultationTypeSelector {...defaultProps} />);

        // Select strategy type
        const strategyCard = screen.getByText('Strategic Planning').closest('div');
        if (strategyCard) {
            await user.click(strategyCard);
        }

        await waitFor(() => {
            expect(screen.getByText('Strategic planning')).toBeInTheDocument();
            expect(screen.getByText('Goal setting')).toBeInTheDocument();
            expect(screen.getByText('Implementation roadmaps')).toBeInTheDocument();
        });
    });

    it('handles keyboard navigation', async () => {
        const user = userEvent.setup();
        render(<ConsultationTypeSelector {...defaultProps} />);

        // Tab to first card and press Enter
        await user.tab();
        await user.keyboard('{Enter}');

        await waitFor(() => {
            expect(screen.getByText('Consultation Details')).toBeInTheDocument();
        });
    });
});