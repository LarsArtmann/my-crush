# MemU Memory Tools

MemU (Memory Unit) provides persistent memory capabilities for AI agents. These tools allow you to:

- **Store information** in memory for future reference
- **Retrieve relevant memories** based on queries  
- **Search memories** for specific information
- **Remove outdated or incorrect memories**

## Usage Examples

### Memorize Important Information
```
Store project documentation, key decisions, or important code patterns:
memu_memorize(resource_url="./docs/api-design.md", modality="text")
memu_memorize(resource_url="conversation-log.json", modality="conversation")
```

### Retrieve Context-Aware Memories
```
Get relevant memories for current task:
memu_retrieve(queries=["authentication patterns", "error handling"])
```

### Search Specific Information
```
Find memories about specific topics:
memu_search(query="database connection issues and solutions")
```

### Forget Outdated Information
```
Remove deprecated or incorrect memories:
memu_forget(query="old authentication method")
```

## Configuration

MemU tools require configuration in your crush.json:

```json
{
  "memu": {
    "enabled": true,
    "data_dir": ".crush/memory",
    "retrieval_mode": "rag"
  }
}
```

- **enabled**: Enable/disable MemU functionality
- **data_dir**: Directory to store memory data
- **retrieval_mode**: "rag" for fast embedding search, "llm" for semantic understanding

## Memory Types

MemU supports multiple content modalities:
- **text**: Documentation, code, notes
- **conversation**: Chat logs, meeting transcripts  
- **log**: Application logs, debug output
- **image**: Visual information, screenshots
- **audio**: Voice notes, recordings
- **video**: Screen recordings, demos

## Best Practices

1. **Be specific**: Store clear, actionable information
2. **Add context**: Include relevant metadata and relationships
3. **Regular cleanup**: Use `memu_forget` to remove outdated information
4. **Categorize**: Use appropriate modality types for better organization
5. **Query effectively**: Combine broad and specific terms for better retrieval