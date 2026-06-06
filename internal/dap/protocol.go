package dap

import "encoding/json"

type Request struct {
	Seq       int             `json:"seq"`
	Type      string          `json:"type"`
	Command   string          `json:"command"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type Response struct {
	Seq        int             `json:"seq"`
	Type       string          `json:"type"`
	RequestSeq int             `json:"request_seq"`
	Success    bool            `json:"success"`
	Command    string          `json:"command"`
	Message    string          `json:"message,omitempty"`
	Body       json.RawMessage `json:"body,omitempty"`
}

type Event struct {
	Seq   int             `json:"seq"`
	Type  string          `json:"type"`
	Event string          `json:"event"`
	Body  json.RawMessage `json:"body,omitempty"`
}

type Message struct {
	Seq        int             `json:"seq"`
	Type       string          `json:"type"`
	Command    string          `json:"command,omitempty"`
	Event      string          `json:"event,omitempty"`
	RequestSeq int             `json:"request_seq,omitempty"`
	Success    bool            `json:"success,omitempty"`
	Message    string          `json:"message,omitempty"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
	Body       json.RawMessage `json:"body,omitempty"`
}

func (m *Message) IsResponse() bool { return m.Type == "response" }
func (m *Message) IsEvent() bool    { return m.Type == "event" }
func (m *Message) IsRequest() bool  { return m.Type == "request" }

// Initialize

type InitializeRequestArguments struct {
	ClientID                     string `json:"clientID,omitempty"`
	ClientName                   string `json:"clientName,omitempty"`
	AdapterID                    string `json:"adapterID"`
	Locale                       string `json:"locale,omitempty"`
	LinesStartAt1                bool   `json:"linesStartAt1"`
	ColumnsStartAt1              bool   `json:"columnsStartAt1"`
	SupportsRunInTerminalRequest bool   `json:"supportsRunInTerminalRequest,omitempty"`
}

type Capabilities struct {
	SupportsConfigurationDoneRequest   bool `json:"supportsConfigurationDoneRequest,omitempty"`
	SupportsFunctionBreakpoints        bool `json:"supportsFunctionBreakpoints,omitempty"`
	SupportsConditionalBreakpoints     bool `json:"supportsConditionalBreakpoints,omitempty"`
	SupportsHitConditionalBreakpoints  bool `json:"supportsHitConditionalBreakpoints,omitempty"`
	SupportsEvaluateForHovers          bool `json:"supportsEvaluateForHovers,omitempty"`
	SupportsSetVariable                bool `json:"supportsSetVariable,omitempty"`
	SupportsLogPoints                  bool `json:"supportsLogPoints,omitempty"`
	SupportsTerminateRequest           bool `json:"supportsTerminateRequest,omitempty"`
	SupportsDataBreakpoints            bool `json:"supportsDataBreakpoints,omitempty"`
	SupportsCancelRequest              bool `json:"supportsCancelRequest,omitempty"`
	SupportsBreakpointLocationsRequest bool `json:"supportsBreakpointLocationsRequest,omitempty"`
	SupportsStepBack                   bool `json:"supportsStepBack,omitempty"`
	SupportsRestartFrame               bool `json:"supportsRestartFrame,omitempty"`
	SupportsCompletionsRequest         bool `json:"supportsCompletionsRequest,omitempty"`
	SupportsRestartRequest             bool `json:"supportsRestartRequest,omitempty"`
	SupportsRunInTerminalRequest       bool `json:"supportsRunInTerminalRequest,omitempty"`
	SupportsProgressReporting          bool `json:"supportsProgressReporting,omitempty"`
	SupportsExceptionInfoRequest       bool `json:"supportsExceptionInfoRequest,omitempty"`

	ExceptionBreakpointFilters []ExceptionBreakpointsFilter `json:"exceptionBreakpointFilters,omitempty"`
}

type ExceptionBreakpointsFilter struct {
	Filter  string `json:"filter"`
	Label   string `json:"label"`
	Default bool   `json:"default,omitempty"`
}

// Launch / Attach

type LaunchRequestArguments struct {
	NoDebug bool   `json:"noDebug,omitempty"`
	Program string `json:"program,omitempty"`

	// Adapter-specific fields passed through as-is.
	Extra map[string]any `json:"-"`
}

type AttachRequestArguments struct {
	Extra map[string]any `json:"-"`
}

// Disconnect

type DisconnectArguments struct {
	Restart          bool `json:"restart,omitempty"`
	TerminateDebuggee bool `json:"terminateDebuggee,omitempty"`
	SuspendDebuggee  bool `json:"suspendDebuggee,omitempty"`
}

// Breakpoints

type Source struct {
	Name            string `json:"name,omitempty"`
	Path            string `json:"path,omitempty"`
	SourceReference int    `json:"sourceReference,omitempty"`
}

type SourceBreakpoint struct {
	Line         int    `json:"line"`
	Column       int    `json:"column,omitempty"`
	Condition    string `json:"condition,omitempty"`
	HitCondition string `json:"hitCondition,omitempty"`
	LogMessage   string `json:"logMessage,omitempty"`
}

type Breakpoint struct {
	ID       int    `json:"id,omitempty"`
	Verified bool   `json:"verified"`
	Message  string `json:"message,omitempty"`
	Source   Source `json:"source,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

type SetBreakpointsArguments struct {
	Source         Source             `json:"source"`
	Breakpoints    []SourceBreakpoint `json:"breakpoints,omitempty"`
	SourceModified bool               `json:"sourceModified,omitempty"`
}

type SetBreakpointsResponseBody struct {
	Breakpoints []Breakpoint `json:"breakpoints"`
}

// Execution control

type ContinueArguments struct {
	ThreadID     int  `json:"threadId"`
	SingleThread bool `json:"singleThread,omitempty"`
}

type ContinueResponseBody struct {
	AllThreadsContinued bool `json:"allThreadsContinued,omitempty"`
}

type StepArguments struct {
	ThreadID     int    `json:"threadId"`
	SingleThread bool   `json:"singleThread,omitempty"`
	Granularity  string `json:"granularity,omitempty"`
}

type PauseArguments struct {
	ThreadID int `json:"threadId"`
}

// Threads

type Thread struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ThreadsResponseBody struct {
	Threads []Thread `json:"threads"`
}

// Stack trace

type StackTraceArguments struct {
	ThreadID   int `json:"threadId"`
	StartFrame int `json:"startFrame,omitempty"`
	Levels     int `json:"levels,omitempty"`
}

type StackFrame struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Source *Source `json:"source,omitempty"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type StackTraceResponseBody struct {
	StackFrames []StackFrame `json:"stackFrames"`
	TotalFrames int          `json:"totalFrames,omitempty"`
}

// Scopes

type ScopesArguments struct {
	FrameID int `json:"frameId"`
}

type Scope struct {
	Name               string `json:"name"`
	PresentationHint   string `json:"presentationHint,omitempty"`
	VariablesReference int    `json:"variablesReference"`
	NamedVariables     int    `json:"namedVariables,omitempty"`
	IndexedVariables   int    `json:"indexedVariables,omitempty"`
	Expensive          bool   `json:"expensive,omitempty"`
}

type ScopesResponseBody struct {
	Scopes []Scope `json:"scopes"`
}

// Variables

type VariablesArguments struct {
	VariablesReference int    `json:"variablesReference"`
	Filter             string `json:"filter,omitempty"`
	Start              int    `json:"start,omitempty"`
	Count              int    `json:"count,omitempty"`
}

type Variable struct {
	Name               string `json:"name"`
	Value              string `json:"value"`
	Type               string `json:"type,omitempty"`
	EvaluateName       string `json:"evaluateName,omitempty"`
	VariablesReference int    `json:"variablesReference"`
	NamedVariables     int    `json:"namedVariables,omitempty"`
	IndexedVariables   int    `json:"indexedVariables,omitempty"`
}

type VariablesResponseBody struct {
	Variables []Variable `json:"variables"`
}

// Evaluate

type EvaluateArguments struct {
	Expression string `json:"expression"`
	FrameID    int    `json:"frameId,omitempty"`
	Context    string `json:"context,omitempty"`
}

type EvaluateResponseBody struct {
	Result             string `json:"result"`
	Type               string `json:"type,omitempty"`
	VariablesReference int    `json:"variablesReference"`
}

// Events

type StoppedEventBody struct {
	Reason            string `json:"reason"`
	Description       string `json:"description,omitempty"`
	ThreadID          int    `json:"threadId,omitempty"`
	PreserveFocusHint bool   `json:"preserveFocusHint,omitempty"`
	Text              string `json:"text,omitempty"`
	AllThreadsStopped bool   `json:"allThreadsStopped,omitempty"`
	HitBreakpointIDs  []int  `json:"hitBreakpointIds,omitempty"`
}

type ContinuedEventBody struct {
	ThreadID            int  `json:"threadId"`
	AllThreadsContinued bool `json:"allThreadsContinued,omitempty"`
}

type ExitedEventBody struct {
	ExitCode int `json:"exitCode"`
}

type TerminatedEventBody struct {
	Restart bool `json:"restart,omitempty"`
}

type ThreadEventBody struct {
	Reason   string `json:"reason"`
	ThreadID int    `json:"threadId"`
}

type OutputEventBody struct {
	Category string `json:"category,omitempty"`
	Output   string `json:"output"`
	Source   *Source `json:"source,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

type BreakpointEventBody struct {
	Reason     string     `json:"reason"`
	Breakpoint Breakpoint `json:"breakpoint"`
}
