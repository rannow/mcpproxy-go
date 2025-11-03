"""Context pruning for conversation history management.

Implements the same strategy as Go LLMAgent:
1. Keep system message and most recent messages
2. Summarize middle messages
3. Remove detailed tool call data from older messages
4. Preserve important context markers (config changes, errors)
"""

import logging
from typing import List, Dict

logger = logging.getLogger(__name__)


class ContextPruner:
    """Intelligent conversation context pruning."""

    # Constants matching Go LLMAgent
    MAX_TOKENS = 100000  # Maximum token budget
    SYSTEM_MESSAGE_TOKENS = 1000  # Reserve for system message
    RECENT_MESSAGE_COUNT = 5  # Always keep the last N messages

    def __init__(self, max_tokens: int = MAX_TOKENS):
        """
        Initialize context pruner.

        Args:
            max_tokens: Maximum token budget for conversation history
        """
        self.max_tokens = max_tokens

    def prune_conversation_history(
        self, messages: List[Dict]
    ) -> List[Dict]:
        """
        Intelligently reduce conversation history to fit token limits.

        Strategy (matching Go LLMAgent):
        1. Keep system message and most recent messages
        2. Summarize middle messages
        3. Remove detailed tool call data from older messages
        4. Preserve important context markers (config changes, errors)

        Args:
            messages: Full conversation history

        Returns:
            Pruned conversation history
        """
        if not messages:
            return []

        # Calculate tokens for each message
        message_tokens = [self._estimate_tokens(msg) for msg in messages]
        total_tokens = sum(message_tokens)

        # No pruning needed if under limit
        if total_tokens <= self.max_tokens:
            logger.debug(
                f"No pruning needed: {total_tokens} tokens < {self.max_tokens} limit"
            )
            return messages

        logger.info(
            f"Context pruning required: {total_tokens} tokens > {self.max_tokens} limit"
        )

        # Start with empty pruned list
        pruned_messages: List[Dict] = []
        current_tokens = 0

        # 1. Keep the first message if it's important context (system message)
        if messages and self._is_system_message(messages[0]):
            pruned_messages.append(messages[0])
            current_tokens += message_tokens[0]
            logger.debug(f"Kept system message: {message_tokens[0]} tokens")

        # 2. Calculate how many middle messages we can afford
        # Reserve space for recent messages
        recent_start_idx = max(1, len(messages) - self.RECENT_MESSAGE_COUNT)
        recent_tokens = sum(message_tokens[recent_start_idx:])

        available_for_middle = (
            self.max_tokens - current_tokens - recent_tokens - self.SYSTEM_MESSAGE_TOKENS
        )

        # 3. Process middle messages (summarize or keep important ones)
        if recent_start_idx > 1:
            middle_messages = messages[1:recent_start_idx]
            middle_tokens = message_tokens[1:recent_start_idx]

            # If middle messages fit, keep them all
            if sum(middle_tokens) <= available_for_middle:
                for msg, tokens in zip(middle_messages, middle_tokens):
                    pruned_messages.append(msg)
                    current_tokens += tokens
                logger.debug(
                    f"Kept all {len(middle_messages)} middle messages: {sum(middle_tokens)} tokens"
                )
            else:
                # Need to prune middle messages - keep only important ones
                important_middle = self._filter_important_messages(
                    middle_messages, middle_tokens, available_for_middle
                )

                for msg, tokens in important_middle:
                    pruned_messages.append(msg)
                    current_tokens += tokens

                logger.debug(
                    f"Kept {len(important_middle)}/{len(middle_messages)} important middle messages"
                )

                # Add summary if we skipped messages
                if len(important_middle) < len(middle_messages):
                    summary = self._create_summary(
                        len(middle_messages) - len(important_middle)
                    )
                    pruned_messages.append(summary)
                    current_tokens += self._estimate_tokens(summary)

        # 4. Add recent messages (these are kept in full detail)
        for i in range(recent_start_idx, len(messages)):
            pruned_messages.append(messages[i])
            current_tokens += message_tokens[i]

        # Log pruning results
        new_total_tokens = sum(self._estimate_tokens(msg) for msg in pruned_messages)
        tokens_saved = total_tokens - new_total_tokens

        logger.info(
            f"Context pruning complete: "
            f"{len(messages)} messages → {len(pruned_messages)} messages, "
            f"{total_tokens} tokens → {new_total_tokens} tokens, "
            f"saved {tokens_saved} tokens ({tokens_saved/total_tokens*100:.1f}%)"
        )

        return pruned_messages

    def _estimate_tokens(self, message: Dict) -> int:
        """
        Estimate token count for a message.

        Uses a simple heuristic: ~4 characters per token
        This matches the Go implementation's estimation.

        Args:
            message: Message dictionary

        Returns:
            Estimated token count
        """
        content = str(message.get("content", ""))

        # Count characters in content
        char_count = len(content)

        # Add tokens for role and metadata
        role_tokens = 5  # Fixed overhead for role, metadata

        # Estimate: ~4 characters per token (English text average)
        estimated_tokens = (char_count // 4) + role_tokens

        return max(estimated_tokens, 10)  # Minimum 10 tokens per message

    def _is_system_message(self, message: Dict) -> bool:
        """
        Check if message is a system message.

        Args:
            message: Message dictionary

        Returns:
            True if system message
        """
        return message.get("role") == "system"

    def _is_important_message(self, message: Dict) -> bool:
        """
        Check if message contains important context markers.

        Important markers (matching Go LLMAgent):
        - Error messages
        - Configuration changes
        - Server status changes
        - Warnings

        Args:
            message: Message dictionary

        Returns:
            True if message is important
        """
        content = str(message.get("content", "")).lower()

        # Important keywords (matching Go implementation)
        important_keywords = [
            "error",
            "failed",
            "warning",
            "critical",
            "configuration",
            "config",
            "server",
            "status",
            "changed",
            "updated",
        ]

        return any(keyword in content for keyword in important_keywords)

    def _filter_important_messages(
        self,
        messages: List[Dict],
        message_tokens: List[int],
        available_tokens: int
    ) -> List[tuple[Dict, int]]:
        """
        Filter middle messages to keep only important ones within token budget.

        Args:
            messages: Middle messages to filter
            message_tokens: Token counts for each message
            available_tokens: Available token budget

        Returns:
            List of (message, tokens) tuples for important messages
        """
        important: List[tuple[Dict, int]] = []
        current_tokens = 0

        for msg, tokens in zip(messages, message_tokens):
            # Keep if important and fits in budget
            if self._is_important_message(msg):
                if current_tokens + tokens <= available_tokens:
                    important.append((msg, tokens))
                    current_tokens += tokens
                else:
                    # Budget exceeded, stop adding
                    break

        return important

    def _create_summary(self, skipped_count: int) -> Dict:
        """
        Create a summary message for skipped middle messages.

        Args:
            skipped_count: Number of messages that were skipped

        Returns:
            Summary message dictionary
        """
        return {
            "role": "assistant",
            "content": f"[Context summary: {skipped_count} older messages omitted to manage context size]"
        }


def prune_if_needed(
    messages: List[Dict],
    max_tokens: int = ContextPruner.MAX_TOKENS
) -> List[Dict]:
    """
    Convenience function to prune conversation history if needed.

    Args:
        messages: Full conversation history
        max_tokens: Maximum token budget (default: 100K)

    Returns:
        Pruned conversation history
    """
    pruner = ContextPruner(max_tokens=max_tokens)
    return pruner.prune_conversation_history(messages)
