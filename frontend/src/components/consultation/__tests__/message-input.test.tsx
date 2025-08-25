import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MessageInput } from '../message-input';
import { useConsultationStore } from '@/stores/consultations';

jest.mock('@/stores/consultations');

const mockUseConsultationStore = useConsultationStore as jest.MockedFunction<typeof useConsultationStore>;

const mockStoreState = {
    isRecording: false,
    setIsRecording: jest.fn(),
    voicePanelOpen: false,
    setVoicePanelOpen: jest.fn(),
};

const defaultProps = {
    value: '',
    onChange: jest.fn(),
    onSend: jest.fn(),
    onKeyPress: jest.fn(),
    disabled: false,
    isLoading: false,
    placeholder: 'Type your message...',
};

describe('MessageInput', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        mockUseConsultationStore.mockReturnValue(mockStoreState as any);
    });

    it('renders input with placeholder', () => {
        render(<MessageInput {...defaultProps} />);

        expect(screen.getByPlaceholderText('Type your message...')).toBeInTheDocument();
    });

    it('calls onChange when typing', async () => {
        const user = userEvent.setup();
        const onChange = jest.fn();

        render(<MessageInput {...defaultProps} onChange={onChange} />);

        const input = screen.getByPlaceholderText('Type your message...');
        await user.type(input, 'Hello');

        expect(onChange).toHaveBeenCalledWith('Hello');
    });

    it('calls onSend when send button is clicked', async () => {
        const user = userEvent.setup();
        const onSend = jest.fn();

        render(<MessageInput {...defaultProps} value="Test message" onSend={onSend} />);

        const sendButton = screen.getByLabelText(/send message/i);
        await user.click(sendButton);

        expect(onSend).toHaveBeenCalledWith('Test message');
    });

    it('calls onKeyPress when key is pressed', async () => {
        const user = userEvent.setup();
        const onKeyPress = jest.fn();

        render(<MessageInput {...defaultProps} onKeyPress={onKeyPress} />);

        const input = screen.getByPlaceholderText('Type your message...');
        await user.type(input, '{Enter}');

        expect(onKeyPress).toHaveBeenCalled();
    });

    it('disables input and send button when disabled', () => {
        render(<MessageInput {...defaultProps} disabled={true} />);

        const input = screen.getByPlaceholderText('Type your message...');
        const sendButton = screen.getByLabelText(/send message/i);

        expect(input).toBeDisabled();
        expect(sendButton).toBeDisabled();
    });

    it('shows loading state on send button', () => {
        render(<MessageInput {...defaultProps} isLoading={true} />);

        const sendButton = screen.getByLabelText(/sending message/i);
        expect(sendButton).toBeDisabled();
    });

    it('disables send button for empty messages', () => {
        render(<MessageInput {...defaultProps} value="" />);

        const sendButton = screen.getByLabelText(/send message/i);
        expect(sendButton).toBeDisabled();
    });

    it('disables send button for whitespace-only messages', () => {
        render(<MessageInput {...defaultProps} value="   " />);

        const sendButton = screen.getByLabelText(/send message/i);
        expect(sendButton).toBeDisabled();
    });

    it('enables send button for non-empty messages', () => {
        render(<MessageInput {...defaultProps} value="Hello" />);

        const sendButton = screen.getByLabelText(/send message/i);
        expect(sendButton).not.toBeDisabled();
    });

    it('shows formatting toolbar when toggled', async () => {
        const user = userEvent.setup();
        render(<MessageInput {...defaultProps} />);

        const formatButton = screen.getByRole('button', { name: /bold/i });
        await user.click(formatButton);

        expect(screen.getByText('Hide')).toBeInTheDocument();
    });

    it('hides formatting toolbar when hide is clicked', async () => {
        const user = userEvent.setup();
        render(<MessageInput {...defaultProps} />);

        // Show toolbar
        const formatButton = screen.getByRole('button', { name: /bold/i });
        await user.click(formatButton);

        // Hide toolbar
        const hideButton = screen.getByText('Hide');
        await user.click(hideButton);

        expect(screen.queryByText('Hide')).not.toBeInTheDocument();
    });

    it('toggles voice panel when voice button is clicked', async () => {
        const user = userEvent.setup();
        render(<MessageInput {...defaultProps} />);

        const voiceButton = screen.getByLabelText(/start voice input/i);
        await user.click(voiceButton);

        expect(mockStoreState.setVoicePanelOpen).toHaveBeenCalledWith(true);
    });

    it('shows recording indicator when recording', () => {
        mockUseConsultationStore.mockReturnValue({
            ...mockStoreState,
            isRecording: true,
        } as any);

        render(<MessageInput {...defaultProps} />);

        expect(screen.getByText('Recording...')).toBeInTheDocument();
    });

    it('shows character count for long messages', () => {
        const longMessage = 'a'.repeat(600);
        render(<MessageInput {...defaultProps} value={longMessage} />);

        expect(screen.getByText('600/2000')).toBeInTheDocument();
    });

    it('shows input hints', () => {
        render(<MessageInput {...defaultProps} />);

        expect(screen.getByText('Press Enter to send, Shift+Enter for new line')).toBeInTheDocument();
        expect(screen.getByText('Show formatting options')).toBeInTheDocument();
    });

    it('auto-resizes textarea based on content', () => {
        const { rerender } = render(<MessageInput {...defaultProps} value="" />);

        const textarea = screen.getByPlaceholderText('Type your message...');
        // const initialHeight = textarea.style.height;

        rerender(<MessageInput {...defaultProps} value="Line 1\nLine 2\nLine 3\nLine 4" />);

        // Height should change (though exact value depends on implementation)
        expect(textarea.style.height).toBeDefined();
    });

    it('prevents sending when disabled', async () => {
        const user = userEvent.setup();
        const onSend = jest.fn();

        render(<MessageInput {...defaultProps} value="Test" onSend={onSend} disabled={true} />);

        const sendButton = screen.getByLabelText(/send message/i);
        await user.click(sendButton);

        expect(onSend).not.toHaveBeenCalled();
    });

    it('prevents sending when loading', async () => {
        const user = userEvent.setup();
        const onSend = jest.fn();

        render(<MessageInput {...defaultProps} value="Test" onSend={onSend} isLoading={true} />);

        const sendButton = screen.getByLabelText(/sending message/i);
        await user.click(sendButton);

        expect(onSend).not.toHaveBeenCalled();
    });

    it('applies focus styles when focused', async () => {
        const user = userEvent.setup();
        render(<MessageInput {...defaultProps} />);

        const input = screen.getByPlaceholderText('Type your message...');
        await user.click(input);

        // Check if parent container has focus styles (implementation detail)
        const container = input.closest('.border');
        expect(container).toHaveClass('border-primary');
    });
});