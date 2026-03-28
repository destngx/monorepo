# Chat Feature - One-Pager

## Overview

The Chat feature provides an interactive AI-powered financial advisor that leverages user data for personalized insights. Users can ask questions about their finances, get investment advice, analyze spending patterns, and receive market intelligence through natural language conversations.

**Primary Use Cases**:

- Real-time financial analysis and advice
- Market research and investment insights
- Spending pattern analysis
- Goal tracking and recommendations
- Transaction categorization assistance

## User Flow

### Basic Conversation Flow

1. **Access**: Navigate to `/chat` page via sidebar or FAB
2. **Model Selection**: Choose AI model (GPT-4o, Claude, Gemini) via model switcher
3. **Query Input**: Type natural language questions about finances
4. **AI Processing**: System analyzes user data, calls financial tools, provides streaming response
5. **Tool Usage**: AI can use web search, data analysis, and financial tools
6. **Follow-up**: Continue conversation with context preservation

### Advanced Features

- **Persistent Chat History**: Conversations saved to localStorage
- **Smart Suggestions**: AI-generated follow-up questions
- **Multi-Modal Responses**: Text, charts, tables, and tool results
- **Context Awareness**: Uses user's financial data for personalized responses

## API Endpoints Used

### `/api/chat` (POST)

**Primary chat endpoint**

- **Parameters**:
  - `messages`: Array of chat messages with roles and content
  - `modelId` (optional): AI model preference (defaults to smart selection)
  - `context` (optional): Previous conversation context
- **Response**: Streaming text response with tool calls
- **Features**: Multi-model support, financial tools, 5-minute timeout
- **Caching**: None (real-time conversation)

### `/api/chat/suggestions` (POST)

**Generates follow-up question suggestions**

- **Parameters**: `context` - Current conversation context
- **Response**: Array of suggested questions with prompts

## Key Components

### Frontend Components

- **ChatContainer**: Main chat interface wrapper
- **ChatInterface**: Message display and input handling
- **ChatMessages**: Individual message rendering with markdown support
- **ChatInput**: Message composition with send button
- **ModelSwitcher**: AI model selection dropdown
- **AIFab**: Floating action button for quick chat access
- **AIDrawer**: Sidebar for chat interface

### State Management

- **useChatMessages**: Manages message state and localStorage persistence
- **useDebouncedChatPersistence**: Debounced saving to prevent excessive writes
- **useAISettings**: Manages AI model preferences and settings

## Data Flow

```
User Input → ChatInterface → useChatMessages → API Call → AI Orchestrator
                                      ↓
                              localStorage Persistence
                                      ↓
                              Streaming Response → UI Updates
```

## Edge Cases & Error Handling

### Rate Limiting

- **Issue**: AI provider rate limits exceeded
- **Handling**: Automatic fallback to alternative models
- **User Experience**: Seamless transition, no interruption

### Context Window Limits

- **Issue**: Long conversations exceed token limits
- **Handling**: Automatic context truncation, conversation summary
- **User Experience**: Transparent continuation

### API Key Issues

- **Issue**: Missing or invalid API keys
- **Handling**: Graceful degradation, clear error messages
- **User Experience**: Helpful setup instructions

### Streaming Timeouts

- **Issue**: Complex queries exceed 5-minute timeout
- **Handling**: Request termination, partial response delivery
- **User Experience**: "Response truncated" notification

### Tool Failures

- **Issue**: Financial tools or web search fails
- **Handling**: Fallback responses without tool data
- **User Experience**: Continues conversation with available information

## Performance Considerations

### Optimization Strategies

- **Debounced Persistence**: 500ms delay on chat history saves
- **Memoized Components**: Message content components prevent re-renders
- **Streaming Responses**: Real-time UI updates without blocking
- **Lazy Loading**: Components loaded on demand

### Resource Usage

- **Memory**: Chat history stored in localStorage (limited by browser)
- **Network**: Streaming responses, tool calls to external APIs
- **CPU**: Markdown rendering, syntax highlighting

### Scalability Limits

- **Message History**: No hard limit, but localStorage constraints apply
- **Concurrent Users**: Per-user isolation, no shared state
- **API Costs**: Usage-based pricing for AI providers

## Configuration Options

### AI Model Selection

- **GPT-4o**: Balanced performance and cost (default for GitHub token)
- **Claude**: Strong reasoning capabilities
- **Gemini**: Google's multimodal model
- **Auto-selection**: Based on available API keys

### Chat Settings

- **Persistence**: Enable/disable localStorage saving
- **Suggestions**: Show/hide AI-generated follow-up questions
- **Streaming**: Enable/disable real-time response streaming

## Testing Considerations

### Unit Tests Needed

- Message formatting and rendering
- localStorage persistence logic
- Model switching functionality
- Error state handling

### Integration Tests Needed

- End-to-end conversation flows
- API error scenarios
- Tool integration testing
- Cross-browser compatibility

### E2E Test Scenarios

- Complete conversation with tool usage
- Error recovery flows
- Model switching during conversation
- Long conversation context management

## Future Enhancements

### Short Term

- Voice input/output support
- Conversation export functionality
- Advanced prompt templates

### Long Term

- Multi-user conversations
- Conversation branching and versioning
- Integration with external financial data sources

---

**Feature Complexity**: High (multi-model AI, streaming, tools)
**User Impact**: Core feature for AI-powered insights
**Technical Debt**: Moderate (complex state management)
**Test Coverage**: Needs expansion</content>
<parameter name="filePath">docs/wealth-management/features/chat.md
