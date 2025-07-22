# Real-World Workflow Examples

Production-ready workflow patterns for common use cases including LLM agents, data pipelines, and API orchestration.

## Available Examples

### 1. LLM Agent Workflow

A think-act-observe loop for an autonomous agent:

```yaml
name: llm-agent
description: Autonomous agent with decision-making capabilities
start: think

nodes:
  - name: think
    type: http
    config:
      url: "https://api.openai.com/v1/chat/completions"
      method: POST
      headers:
        Authorization: "Bearer ${OPENAI_API_KEY}"
        Content-Type: "application/json"
      body:
        model: "gpt-4"
        messages:
          - role: "system"
            content: "You are an assistant that decides what action to take next."
          - role: "user"
            content: "{{.context}}"
        temperature: 0.7
        
  - name: extract-action
    type: jsonpath
    config:
      path: "$.choices[0].message.content"
      
  - name: route-action
    type: conditional
    config:
      conditions:
        - if: '{{contains . "SEARCH:"}}'
          then: search
        - if: '{{contains . "CALCULATE:"}}'
          then: calculate
        - if: '{{contains . "COMPLETE:"}}'
          then: complete
      else: think
      
  - name: search
    type: http
    config:
      url: "https://api.search.com/v1/search"
      method: GET
      params:
        q: "{{.query}}"
        
  - name: calculate
    type: lua
    config:
      script: |
        local expr = input.expression
        -- Safely evaluate mathematical expression
        local result = evaluate_math(expr)
        return {result = result}
        
  - name: complete
    type: template
    config:
      template: |
        Task completed successfully.
        Final result: {{.result}}

connections:
  - from: think
    to: extract-action
  - from: extract-action
    to: route-action
  - from: search
    to: think
  - from: calculate
    to: think
```

### 2. Data Processing Pipeline

ETL pipeline for processing customer data:

```yaml
name: customer-data-pipeline
description: Extract, transform, and load customer data
start: extract

nodes:
  - name: extract
    type: parallel
    config:
      tasks:
        - name: extract-crm
          node: http
          config:
            url: "${CRM_API_URL}/customers"
            headers:
              X-API-Key: "${CRM_API_KEY}"
              
        - name: extract-billing
          node: http
          config:
            url: "${BILLING_API_URL}/accounts"
            headers:
              Authorization: "Bearer ${BILLING_TOKEN}"
              
  - name: merge-data
    type: aggregate
    config:
      mode: merge
      timeout: "30s"
      
  - name: validate
    type: validate
    config:
      schema:
        type: array
        items:
          type: object
          properties:
            customer_id:
              type: string
            email:
              type: string
              format: email
            status:
              type: string
              enum: ["active", "inactive", "pending"]
          required: ["customer_id", "email"]
          
  - name: transform
    type: transform
    config:
      jq: |
        map({
          id: .customer_id,
          email: .email | ascii_downcase,
          status: .status,
          last_updated: now | todate,
          revenue: .billing.total_revenue // 0,
          tier: if .billing.total_revenue > 10000 then "enterprise"
                elif .billing.total_revenue > 1000 then "professional"
                else "starter" end
        })
        
  - name: load
    type: http
    config:
      url: "${WAREHOUSE_API_URL}/customers/batch"
      method: POST
      headers:
        Content-Type: "application/json"
        X-API-Key: "${WAREHOUSE_API_KEY}"
      retry:
        max_attempts: 5
        delay: "5s"
        multiplier: 2

connections:
  - from: extract
    to: merge-data
  - from: merge-data
    to: validate
  - from: validate
    to: transform
    action: valid
  - from: transform
    to: load
```

### 3. Monitoring and Alerting Workflow

Health check and alerting system:

```yaml
name: health-monitoring
description: Monitor services and send alerts
start: check-services

nodes:
  - name: check-services
    type: parallel
    config:
      fail_fast: false
      timeout: "10s"
      tasks:
        - name: check-api
          node: http
          config:
            url: "https://api.example.com/health"
            timeout: "5s"
            
        - name: check-database
          node: exec
          config:
            command: pg_isready
            args: ["-h", "db.example.com", "-p", "5432"]
            timeout: "5s"
            
        - name: check-cache
          node: http
          config:
            url: "http://redis.example.com:6379/ping"
            timeout: "5s"
            
  - name: analyze-results
    type: lua
    config:
      script: |
        local results = input
        local failures = {}
        
        for service, result in pairs(results) do
          if result.status ~= 200 and result.code ~= 0 then
            table.insert(failures, {
              service = service,
              error = result.error or "Unhealthy"
            })
          end
        end
        
        return {
          healthy = #failures == 0,
          failures = failures,
          checked_at = os.time()
        }
        
  - name: route-status
    type: conditional
    config:
      conditions:
        - if: "{{.healthy}}"
          then: log-success
        - if: "{{not .healthy}}"
          then: send-alert
          
  - name: send-alert
    type: parallel
    config:
      tasks:
        - name: slack-alert
          node: http
          config:
            url: "${SLACK_WEBHOOK_URL}"
            method: POST
            body:
              text: "🚨 Service Alert: {{len .failures}} services are down"
              attachments:
                - color: "danger"
                  fields: |
                    {{range .failures}}
                    - title: "{{.service}}"
                      value: "{{.error}}"
                      short: true
                    {{end}}
                    
        - name: pagerduty-alert
          node: http
          config:
            url: "https://events.pagerduty.com/v2/enqueue"
            method: POST
            headers:
              Authorization: "Token token=${PAGERDUTY_TOKEN}"
            body:
              routing_key: "${PAGERDUTY_ROUTING_KEY}"
              event_action: "trigger"
              payload:
                summary: "Multiple services are experiencing issues"
                severity: "error"
                custom_details: "{{.failures}}"
                
  - name: log-success
    type: echo
    config:
      message: "All services healthy at {{.checked_at}}"
```

### 4. Order Processing Workflow

E-commerce order fulfillment pipeline:

```yaml
name: order-processing
description: Process customer orders from validation to fulfillment
start: receive-order

nodes:
  - name: receive-order
    type: validate
    config:
      schema:
        type: object
        properties:
          order_id:
            type: string
            pattern: "^ORD-[0-9]{8}$"
          customer:
            type: object
            properties:
              id:
                type: string
              email:
                type: string
                format: email
          items:
            type: array
            minItems: 1
            items:
              type: object
              properties:
                sku:
                  type: string
                quantity:
                  type: integer
                  minimum: 1
        required: ["order_id", "customer", "items"]
        
  - name: check-inventory
    type: parallel
    config:
      tasks: |
        {{range .items}}
        - name: "check-{{.sku}}"
          node: http
          config:
            url: "${INVENTORY_API}/check"
            method: POST
            body:
              sku: "{{.sku}}"
              quantity: {{.quantity}}
        {{end}}
        
  - name: calculate-pricing
    type: http
    config:
      url: "${PRICING_API}/calculate"
      method: POST
      body:
        items: "{{.items}}"
        customer_id: "{{.customer.id}}"
        
  - name: process-payment
    type: http
    config:
      url: "${PAYMENT_API}/charge"
      method: POST
      body:
        amount: "{{.total}}"
        currency: "USD"
        customer_id: "{{.customer.id}}"
        order_id: "{{.order_id}}"
      timeout: "30s"
      retry:
        max_attempts: 3
        delay: "5s"
        
  - name: create-fulfillment
    type: http
    config:
      url: "${FULFILLMENT_API}/orders"
      method: POST
      body:
        order_id: "{{.order_id}}"
        items: "{{.items}}"
        shipping_address: "{{.customer.shipping_address}}"
        
  - name: send-confirmation
    type: parallel
    config:
      tasks:
        - name: email-confirmation
          node: http
          config:
            url: "${EMAIL_API}/send"
            method: POST
            body:
              to: "{{.customer.email}}"
              template: "order_confirmation"
              data:
                order_id: "{{.order_id}}"
                items: "{{.items}}"
                total: "{{.total}}"
                
        - name: sms-notification
          node: http
          config:
            url: "${SMS_API}/send"
            method: POST
            body:
              to: "{{.customer.phone}}"
              message: "Order {{.order_id}} confirmed! Track at example.com/track/{{.order_id}}"

connections:
  - from: receive-order
    to: check-inventory
    action: valid
  - from: check-inventory
    to: calculate-pricing
  - from: calculate-pricing
    to: process-payment
  - from: process-payment
    to: create-fulfillment
    action: success
  - from: create-fulfillment
    to: send-confirmation
```

## Key Patterns Demonstrated

### LLM Integration
- API authentication
- Response parsing
- Decision routing
- Context management

### Data Pipeline
- Parallel data extraction
- Data merging and transformation
- Schema validation
- Batch loading

### Monitoring
- Health checks
- Alert routing
- Multi-channel notifications
- Failure aggregation

### Business Process
- Order validation
- Inventory checking
- Payment processing
- Multi-step fulfillment

## Best Practices

1. **Environment Variables**: Use `${VAR}` for secrets and configuration
2. **Error Handling**: Always define error paths and retries
3. **Timeouts**: Set appropriate timeouts for external calls
4. **Validation**: Validate data early in the workflow
5. **Idempotency**: Design workflows to be safely re-runnable
6. **Monitoring**: Add logging and metrics collection
7. **Security**: Never hardcode credentials

## Deployment Considerations

### Configuration Management
```yaml
# config/production.yaml
api_endpoints:
  crm: "https://crm.example.com/api/v2"
  billing: "https://billing.example.com/api/v1"
  
timeouts:
  default: "30s"
  payment: "60s"
  
retry:
  max_attempts: 3
  delay: "5s"
  multiplier: 2
```

### Running in Production
```bash
# With environment file
pocket run order-processing.yaml --env-file .env.production

# With config override
pocket run monitoring.yaml --config config/production.yaml

# With input from file
pocket run data-pipeline.yaml --input-file batch-001.json
```

### Monitoring and Logging
```bash
# Enable detailed logging
POCKET_LOG_LEVEL=debug pocket run workflow.yaml

# Output metrics
pocket run workflow.yaml --metrics-port 9090

# Save execution trace
pocket run workflow.yaml --trace-file trace.json
```

## Next Steps

- Learn about [Performance Optimization](../../advanced/PERFORMANCE.md)
- Explore [Custom Node Development](../../advanced/CUSTOM_NODES.md)
- Study [Production Deployment](../../guides/)