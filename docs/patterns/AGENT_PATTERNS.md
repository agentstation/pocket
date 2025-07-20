# Agent Patterns

## Overview

Agents are autonomous entities that observe, think, decide, and act. Pocket's graph-based architecture is ideal for implementing agent patterns, from simple reactive agents to complex cognitive architectures.

## Basic Agent Loop

### Think-Act Pattern

The fundamental agent pattern involves thinking (planning) and acting (execution):

```go
type AgentState struct {
    Goal        string
    Observation any
    Plan        []Action
    History     []Action
}

// Think node - analyzes state and creates plan
think := pocket.NewNode[AgentState, AgentState]("think",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, state AgentState) (any, error) {
        // Gather context from store
        history, _ := store.Get(ctx, "agent:history")
        knowledge, _ := store.Get(ctx, "agent:knowledge")
        
        return map[string]any{
            "state":     state,
            "history":   history,
            "knowledge": knowledge,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (AgentState, error) {
        data := prepData.(map[string]any)
        state := data["state"].(AgentState)
        
        // Analyze current situation
        situation := analyzeSituation(state.Observation, data["history"])
        
        // Create plan
        plan := createPlan(state.Goal, situation, data["knowledge"])
        
        state.Plan = plan
        return state, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter, 
        input AgentState, prep, result any) (AgentState, string, error) {
        
        state := result.(AgentState)
        
        // No plan means goal achieved
        if len(state.Plan) == 0 {
            return state, "complete", nil
        }
        
        // Execute next action
        return state, "act", nil
    }),
)

// Act node - executes planned actions
act := pocket.NewNode[AgentState, AgentState]("act",
    pocket.WithExec(func(ctx context.Context, state AgentState) (AgentState, error) {
        if len(state.Plan) == 0 {
            return state, errors.New("no actions to execute")
        }
        
        // Execute first action
        action := state.Plan[0]
        result, err := executeAction(action)
        if err != nil {
            return state, fmt.Errorf("action failed: %w", err)
        }
        
        // Update state
        state.Observation = result
        state.Plan = state.Plan[1:] // Remove executed action
        state.History = append(state.History, action)
        
        return state, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input AgentState, prep, result any) (AgentState, string, error) {
        
        state := result.(AgentState)
        
        // Save history
        store.Set(ctx, "agent:history", state.History)
        
        // Continue thinking
        return state, "think", nil
    }),
)

// Connect the loop
think.Connect("act", act)
act.Connect("think", think)
think.Connect("complete", completeNode)

// Start the agent
agentGraph := pocket.NewGraph(think, store)
finalState, err := agentGraph.Run(ctx, AgentState{
    Goal: "Complete the task",
})
```

## Reactive Agent

A simple agent that reacts to environmental stimuli:

```go
// Reactive agent with condition-action rules
type ReactiveAgent struct {
    Rules []Rule
}

type Rule struct {
    Condition func(Perception) bool
    Action    string
}

reactiveAgent := pocket.NewNode[Perception, Action]("reactive-agent",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, percept Perception) (any, error) {
        // Load agent rules
        rules, _ := store.Get(ctx, "agent:rules")
        return map[string]any{
            "percept": percept,
            "rules":   rules.([]Rule),
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (Action, error) {
        data := prepData.(map[string]any)
        percept := data["percept"].(Perception)
        rules := data["rules"].([]Rule)
        
        // Find first matching rule
        for _, rule := range rules {
            if rule.Condition(percept) {
                return Action{Type: rule.Action}, nil
            }
        }
        
        // Default action
        return Action{Type: "wait"}, nil
    }),
)

// Usage with environmental loop
environment := pocket.NewNode[Action, Perception]("environment",
    pocket.WithExec(func(ctx context.Context, action Action) (Perception, error) {
        // Apply action to environment
        newState := applyAction(currentEnvironment, action)
        
        // Generate perception
        return generatePerception(newState), nil
    }),
)

// Connect agent to environment
reactiveAgent.Connect("default", environment)
environment.Connect("default", reactiveAgent)
```

## Goal-Oriented Agent

An agent that maintains goals and plans to achieve them:

```go
type Goal struct {
    ID          string
    Description string
    Priority    int
    Status      string
    Conditions  []Condition
}

type GoalOrientedAgent struct {
    Goals       []Goal
    ActiveGoal  *Goal
    Plan        []Step
    Beliefs     map[string]any
}

// Goal selection node
selectGoal := pocket.NewNode[GoalOrientedAgent, GoalOrientedAgent]("select-goal",
    pocket.WithExec(func(ctx context.Context, agent GoalOrientedAgent) (GoalOrientedAgent, error) {
        // Find highest priority achievable goal
        var selectedGoal *Goal
        highestPriority := -1
        
        for i := range agent.Goals {
            goal := &agent.Goals[i]
            if goal.Status == "completed" {
                continue
            }
            
            if isAchievable(goal, agent.Beliefs) && goal.Priority > highestPriority {
                selectedGoal = goal
                highestPriority = goal.Priority
            }
        }
        
        agent.ActiveGoal = selectedGoal
        return agent, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input, prep, result any) (GoalOrientedAgent, string, error) {
        
        agent := result.(GoalOrientedAgent)
        
        if agent.ActiveGoal == nil {
            return agent, "idle", nil
        }
        
        return agent, "plan", nil
    }),
)

// Planning node
planGoal := pocket.NewNode[GoalOrientedAgent, GoalOrientedAgent]("plan-goal",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, agent GoalOrientedAgent) (any, error) {
        // Load planning knowledge
        operators, _ := store.Get(ctx, "planning:operators")
        worldModel, _ := store.Get(ctx, "world:model")
        
        return map[string]any{
            "agent":      agent,
            "operators":  operators,
            "worldModel": worldModel,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (GoalOrientedAgent, error) {
        data := prepData.(map[string]any)
        agent := data["agent"].(GoalOrientedAgent)
        
        // Create plan to achieve goal
        plan := createPlanForGoal(
            agent.ActiveGoal,
            agent.Beliefs,
            data["operators"],
            data["worldModel"],
        )
        
        agent.Plan = plan
        return agent, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input, prep, result any) (GoalOrientedAgent, string, error) {
        
        agent := result.(GoalOrientedAgent)
        
        if len(agent.Plan) == 0 {
            // Could not create plan
            agent.ActiveGoal.Status = "impossible"
            return agent, "select-goal", nil
        }
        
        return agent, "execute", nil
    }),
)

// Execution node
executeStep := pocket.NewNode[GoalOrientedAgent, GoalOrientedAgent]("execute-step",
    pocket.WithExec(func(ctx context.Context, agent GoalOrientedAgent) (GoalOrientedAgent, error) {
        if len(agent.Plan) == 0 {
            return agent, nil
        }
        
        // Execute next step
        step := agent.Plan[0]
        result, err := performStep(step)
        if err != nil {
            // Plan failed
            agent.ActiveGoal.Status = "failed"
            agent.Plan = nil
            return agent, nil
        }
        
        // Update beliefs based on result
        updateBeliefs(&agent.Beliefs, result)
        
        // Remove completed step
        agent.Plan = agent.Plan[1:]
        
        // Check if goal achieved
        if checkGoalConditions(agent.ActiveGoal, agent.Beliefs) {
            agent.ActiveGoal.Status = "completed"
            agent.Plan = nil
        }
        
        return agent, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input, prep, result any) (GoalOrientedAgent, string, error) {
        
        agent := result.(GoalOrientedAgent)
        
        // Save updated beliefs
        store.Set(ctx, "agent:beliefs", agent.Beliefs)
        
        if len(agent.Plan) > 0 {
            // Continue execution
            return agent, "execute", nil
        }
        
        // Select new goal
        return agent, "select-goal", nil
    }),
)
```

## Learning Agent

An agent that improves its behavior based on experience:

```go
type Experience struct {
    State      State
    Action     Action
    Result     Result
    Reward     float64
    NextState  State
}

type LearningAgent struct {
    Policy      map[string]ActionWeights
    Experiences []Experience
    Epsilon     float64 // Exploration rate
}

// Observe and learn node
observeLearn := pocket.NewNode[Observation, LearningAgent]("observe-learn",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, obs Observation) (any, error) {
        agent, _ := store.Get(ctx, "learning:agent")
        lastAction, _ := store.Get(ctx, "learning:lastAction")
        lastState, _ := store.Get(ctx, "learning:lastState")
        
        return map[string]any{
            "observation": obs,
            "agent":       agent.(LearningAgent),
            "lastAction":  lastAction,
            "lastState":   lastState,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (LearningAgent, error) {
        data := prepData.(map[string]any)
        agent := data["agent"].(LearningAgent)
        obs := data["observation"].(Observation)
        
        // Create experience from last action
        if data["lastAction"] != nil && data["lastState"] != nil {
            exp := Experience{
                State:     data["lastState"].(State),
                Action:    data["lastAction"].(Action),
                Result:    obs.Result,
                Reward:    calculateReward(obs),
                NextState: obs.State,
            }
            
            agent.Experiences = append(agent.Experiences, exp)
            
            // Update policy (Q-learning style)
            updatePolicy(&agent.Policy, exp)
        }
        
        return agent, nil
    }),
)

// Decision making node
decide := pocket.NewNode[LearningAgent, ActionDecision]("decide",
    pocket.WithExec(func(ctx context.Context, agent LearningAgent) (ActionDecision, error) {
        currentState := getCurrentState()
        
        // Epsilon-greedy action selection
        if rand.Float64() < agent.Epsilon {
            // Explore: random action
            return ActionDecision{
                Action: selectRandomAction(),
                Type:   "exploration",
            }, nil
        }
        
        // Exploit: best known action
        stateKey := currentState.Key()
        weights := agent.Policy[stateKey]
        bestAction := selectBestAction(weights)
        
        return ActionDecision{
            Action: bestAction,
            Type:   "exploitation",
        }, nil
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        agent LearningAgent, prep, decision any) (ActionDecision, string, error) {
        
        dec := decision.(ActionDecision)
        
        // Save for next learning cycle
        store.Set(ctx, "learning:agent", agent)
        store.Set(ctx, "learning:lastAction", dec.Action)
        store.Set(ctx, "learning:lastState", getCurrentState())
        
        return dec, "execute", nil
    }),
)

// Training loop
func TrainAgent(ctx context.Context, episodes int) (*LearningAgent, error) {
    agent := &LearningAgent{
        Policy:  make(map[string]ActionWeights),
        Epsilon: 0.1, // 10% exploration
    }
    
    store := pocket.NewStore()
    store.Set(ctx, "learning:agent", *agent)
    
    // Build learning graph
    observeLearn.Connect("default", decide)
    decide.Connect("execute", executeAction)
    executeAction.Connect("observe", observeLearn)
    
    graph := pocket.NewGraph(observeLearn, store)
    
    // Run training episodes
    for i := 0; i < episodes; i++ {
        // Decay exploration over time
        agent.Epsilon = 0.1 * (1.0 - float64(i)/float64(episodes))
        
        // Run episode
        _, err := graph.Run(ctx, getInitialObservation())
        if err != nil {
            return nil, err
        }
        
        // Periodic learning from batch
        if i%100 == 0 {
            batchLearn(agent)
        }
    }
    
    return agent, nil
}
```

## Multi-Agent System

Multiple agents interacting in a shared environment:

```go
type MultiAgentSystem struct {
    Agents      map[string]Agent
    Environment *Environment
    Messages    chan Message
}

type Message struct {
    From    string
    To      string
    Type    string
    Content any
}

// Agent communication node
communicate := pocket.NewNode[AgentMessage, CommunicationResult]("communicate",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, msg AgentMessage) (any, error) {
        // Get agent roster
        agents, _ := store.Get(ctx, "mas:agents")
        messageQueue, _ := store.Get(ctx, "mas:messages")
        
        return map[string]any{
            "message":      msg,
            "agents":       agents,
            "messageQueue": messageQueue,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (CommunicationResult, error) {
        data := prepData.(map[string]any)
        msg := data["message"].(AgentMessage)
        
        switch msg.Type {
        case "broadcast":
            // Send to all agents
            return broadcastMessage(msg, data["agents"])
            
        case "negotiate":
            // Start negotiation protocol
            return initiateNegotiation(msg, data["agents"])
            
        case "collaborate":
            // Form collaboration
            return formCollaboration(msg, data["agents"])
            
        default:
            // Direct message
            return sendDirectMessage(msg, data["agents"])
        }
    }),
    pocket.WithPost(func(ctx context.Context, store pocket.StoreWriter,
        input, prep, result any) (CommunicationResult, string, error) {
        
        res := result.(CommunicationResult)
        
        // Update message queue
        queue, _ := store.Get(ctx, "mas:messages")
        updatedQueue := append(queue.([]Message), res.SentMessages...)
        store.Set(ctx, "mas:messages", updatedQueue)
        
        return res, "process-responses", nil
    }),
)

// Coordination node
coordinate := pocket.NewNode[CoordinationRequest, CoordinationPlan]("coordinate",
    pocket.WithExec(func(ctx context.Context, req CoordinationRequest) (CoordinationPlan, error) {
        // Analyze task requirements
        taskAnalysis := analyzeTask(req.Task)
        
        // Select agents based on capabilities
        selectedAgents := selectAgents(req.AvailableAgents, taskAnalysis.RequiredCapabilities)
        
        // Create coordination plan
        plan := CoordinationPlan{
            Task:   req.Task,
            Agents: selectedAgents,
            Roles:  assignRoles(selectedAgents, taskAnalysis),
            Schedule: createSchedule(taskAnalysis.Subtasks, selectedAgents),
        }
        
        return plan, nil
    }),
)

// Conflict resolution node
resolveConflict := pocket.NewNode[Conflict, Resolution]("resolve-conflict",
    pocket.WithExec(func(ctx context.Context, conflict Conflict) (Resolution, error) {
        switch conflict.Type {
        case "resource":
            return resolveResourceConflict(conflict)
            
        case "goal":
            return resolveGoalConflict(conflict)
            
        case "plan":
            return resolvePlanConflict(conflict)
            
        default:
            return negotiateResolution(conflict)
        }
    }),
)
```

## Cognitive Architecture

A more sophisticated agent with memory, reasoning, and learning:

```go
type CognitiveAgent struct {
    // Memory systems
    ShortTermMemory []Memory
    LongTermMemory  *MemoryStore
    WorkingMemory   map[string]any
    
    // Cognitive processes
    Attention    *AttentionSystem
    Reasoning    *ReasoningEngine
    Learning     *LearningSystem
    
    // Current state
    Goals        []Goal
    Beliefs      BeliefSet
    Emotions     EmotionalState
}

// Perception and attention
perceive := pocket.NewNode[SensoryInput, Perception]("perceive",
    pocket.WithExec(func(ctx context.Context, input SensoryInput) (Perception, error) {
        // Filter through attention system
        filtered := attentionFilter(input)
        
        // Pattern recognition
        patterns := recognizePatterns(filtered)
        
        // Create perception
        return Perception{
            Timestamp: time.Now(),
            Patterns:  patterns,
            Salience:  calculateSalience(patterns),
        }, nil
    }),
)

// Memory consolidation
consolidateMemory := pocket.NewNode[CognitiveAgent, CognitiveAgent]("consolidate-memory",
    pocket.WithExec(func(ctx context.Context, agent CognitiveAgent) (CognitiveAgent, error) {
        // Move important short-term memories to long-term
        for _, memory := range agent.ShortTermMemory {
            if memory.Importance > ImportanceThreshold {
                agent.LongTermMemory.Store(memory)
            }
        }
        
        // Decay old short-term memories
        agent.ShortTermMemory = decayMemories(agent.ShortTermMemory)
        
        // Consolidate patterns
        patterns := extractPatterns(agent.ShortTermMemory)
        for _, pattern := range patterns {
            agent.LongTermMemory.StorePattern(pattern)
        }
        
        return agent, nil
    }),
)

// Reasoning and decision making
reason := pocket.NewNode[CognitiveAgent, Decision]("reason",
    pocket.WithPrep(func(ctx context.Context, store pocket.StoreReader, agent CognitiveAgent) (any, error) {
        // Load relevant memories
        context := buildReasoningContext(agent)
        
        return map[string]any{
            "agent":   agent,
            "context": context,
        }, nil
    }),
    pocket.WithExec(func(ctx context.Context, prepData any) (Decision, error) {
        data := prepData.(map[string]any)
        agent := data["agent"].(CognitiveAgent)
        context := data["context"].(ReasoningContext)
        
        // Generate hypotheses
        hypotheses := generateHypotheses(agent.Beliefs, context)
        
        // Evaluate each hypothesis
        evaluations := make([]Evaluation, len(hypotheses))
        for i, hypothesis := range hypotheses {
            evaluations[i] = evaluateHypothesis(hypothesis, agent, context)
        }
        
        // Select best action
        bestAction := selectBestAction(evaluations, agent.Goals, agent.Emotions)
        
        return Decision{
            Action:     bestAction,
            Confidence: calculateConfidence(evaluations),
            Reasoning:  explainReasoning(bestAction, evaluations),
        }, nil
    }),
)

// Emotional processing
processEmotions := pocket.NewNode[CognitiveAgent, CognitiveAgent]("process-emotions",
    pocket.WithExec(func(ctx context.Context, agent CognitiveAgent) (CognitiveAgent, error) {
        // Update emotional state based on events
        events := getRecentEvents(agent.ShortTermMemory)
        
        for _, event := range events {
            emotionalImpact := assessEmotionalImpact(event, agent.Beliefs, agent.Goals)
            agent.Emotions = updateEmotionalState(agent.Emotions, emotionalImpact)
        }
        
        // Emotional regulation
        if agent.Emotions.Intensity > RegulationThreshold {
            agent.Emotions = regulateEmotions(agent.Emotions)
        }
        
        return agent, nil
    }),
)

// Learning and adaptation
learn := pocket.NewNode[Experience, LearningUpdate]("learn",
    pocket.WithExec(func(ctx context.Context, exp Experience) (LearningUpdate, error) {
        // Extract lessons
        lessons := extractLessons(exp)
        
        // Update models
        updates := LearningUpdate{
            BeliefUpdates: updateBeliefs(exp, lessons),
            SkillUpdates:  updateSkills(exp, lessons),
            GoalUpdates:   updateGoals(exp, lessons),
        }
        
        return updates, nil
    }),
)
```

## Agent Patterns Best Practices

### 1. State Management

```go
// Maintain clean agent state
type CleanAgentState struct {
    // Immutable identity
    ID       string
    Type     string
    
    // Mutable state with versioning
    Version  int
    State    map[string]any
    
    // Separate concerns
    Percepts []Percept     // Input
    Actions  []Action      // Output
    Internal InternalState // Processing
}
```

### 2. Modular Behaviors

```go
// Compose behaviors from reusable components
type Behavior interface {
    Evaluate(agent Agent, context Context) float64
    Execute(agent Agent) (Action, error)
}

type CompositeBehavior struct {
    Behaviors []Behavior
    Selector  SelectionStrategy
}
```

### 3. Testable Agents

```go
func TestAgentBehavior(t *testing.T) {
    // Create test environment
    env := NewTestEnvironment()
    agent := NewAgent(testConfig)
    
    // Define scenario
    scenario := Scenario{
        InitialState: State{...},
        Events: []Event{...},
        ExpectedActions: []Action{...},
    }
    
    // Run agent
    actions := runAgentInScenario(agent, env, scenario)
    
    // Verify behavior
    assert.Equal(t, scenario.ExpectedActions, actions)
}
```

### 4. Observable Agents

```go
// Make agent behavior observable
type ObservableAgent struct {
    Agent
    observers []Observer
}

func (a *ObservableAgent) Notify(event AgentEvent) {
    for _, observer := range a.observers {
        observer.OnAgentEvent(event)
    }
}
```

## Summary

Agent patterns in Pocket enable:

1. **Autonomous behavior** through think-act loops
2. **Goal-oriented planning** and execution
3. **Learning and adaptation** from experience
4. **Multi-agent coordination** and communication
5. **Cognitive architectures** with memory and reasoning

These patterns provide the foundation for building intelligent, autonomous systems that can perceive, reason, learn, and act in complex environments.