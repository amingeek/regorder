import os
import sys
from llama_cpp import Llama

MODEL_PATH = "./whisper.cpp/models/mpt-3b-8k-instruct.q2_k.gguf"
HISTORY_FILE = "./history.txt"

# Load history if exists
if os.path.exists(HISTORY_FILE):
    with open(HISTORY_FILE, "r", encoding="utf-8") as f:
        history_text = f.read().strip()
else:
    history_text = ""

# User input
if len(sys.argv) < 2:
    print("Usage: python3 llm_cpu_model.py '<user message>'")
    sys.exit(1)

user_message = sys.argv[1].strip()

# Update history with user message
history_text += f"\nUser: {user_message}\n"

# Initialize LLaMA/MPT model
llm = Llama(
    model_path=MODEL_PATH,
    n_ctx=2048,
    n_threads=os.cpu_count(),
)

# Prepare prompt with full history
prompt = f"""
You are a helpful assistant. Keep the conversation context from the user history.
Respond only in English.

Conversation history:
{history_text}
Assistant:
"""

# Generate response
resp = llm(prompt, max_tokens=256, stop=["User:", "Assistant:"])
assistant_text = resp['choices'][0]['text'].strip()

# Print assistant response
print(assistant_text)

# Append assistant response to history
history_text += f"Assistant: {assistant_text}\n"

# Save updated history
with open(HISTORY_FILE, "w", encoding="utf-8") as f:
    f.write(history_text)
