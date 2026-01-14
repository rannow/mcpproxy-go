# MCP Server Complete Test Results

**Date:** $(date '+%Y-%m-%d %H:%M:%S')
**Method:** Full Functional Testing with Real Data
**API Endpoint:** /chat/call-tool

---

## Executive Summary

| Metric | Value |
|--------|-------|
| Total Servers | 43 |
| Total Tools | 329 |
| Tests Passed | 215 |
| Tests Failed | 86 (+ 28 warnings) |
| Success Rate | 65.3% |

---

## Detailed Results by Server


### archon
**Status:** ERROR - Could not retrieve tools


### calculator
**Tools:** 1

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | calculate | ✅ PASS | 192ms | Validation OK |

**Summary:** 1 passed, 0 failed

---

### claude-code-mcp
**Tools:** 1

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | claude_code | ❌ FAIL | 189ms | Failed to call tool: tool 'claude_code'  |

**Summary:** 0 passed, 1 failed

---

### confluence
**Tools:** 5

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | conf_get | ⚠️ WARN | 178ms | MCP error -32602: Input validation error |
| 2 | conf_post | ⚠️ WARN | 189ms | MCP error -32602: Input validation error |
| 3 | conf_put | ⚠️ WARN | 192ms | MCP error -32602: Input validation error |
| 4 | conf_patch | ⚠️ WARN | 177ms | MCP error -32602: Input validation error |
| 5 | conf_delete | ⚠️ WARN | 234ms | MCP error -32602: Input validation error |

**Summary:** 0 passed, 0 failed

---

### context7
**Tools:** 2

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | resolve-library-id | ✅ PASS | 208ms | Validation OK |
| 2 | query-docs | ✅ PASS | 248ms | Validation OK |

**Summary:** 2 passed, 0 failed

---

### desktop-commander
**Status:** ERROR - Could not retrieve tools


### docker
**Tools:** 1

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | run_command | ❌ FAIL | 253ms | Failed to call tool: tool 'run_command'  |

**Summary:** 0 passed, 1 failed

---

### exa
**Tools:** 2

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | web_search_exa | ✅ PASS | 1066ms | Validation OK |
| 2 | get_code_context_exa | ✅ PASS | 248ms | Validation OK |

**Summary:** 2 passed, 0 failed

---

### excel
**Tools:** 6

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | excel_copy_sheet | ✅ PASS | 235ms | Validation OK |
| 2 | excel_create_table | ✅ PASS | 242ms | Validation OK |
| 3 | excel_describe_sheets | ✅ PASS | 241ms | Validation OK |
| 4 | excel_format_range | ✅ PASS | 237ms | Validation OK |
| 5 | excel_read_sheet | ✅ PASS | 276ms | Validation OK |
| 6 | excel_write_to_sheet | ✅ PASS | 256ms | Validation OK |

**Summary:** 6 passed, 0 failed

---

### excel-advanced
**Tools:** 25

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | apply_formula | ✅ PASS | 231ms | Validation OK |
| 2 | validate_formula_syntax | ✅ PASS | 238ms | Validation OK |
| 3 | format_range | ✅ PASS | 271ms | Validation OK |
| 4 | read_data_from_excel | ✅ PASS | 261ms | Validation OK |
| 5 | write_data_to_excel | ✅ PASS | 235ms | Validation OK |
| 6 | create_workbook | ✅ PASS | 238ms | Validation OK |
| 7 | create_worksheet | ✅ PASS | 245ms | Validation OK |
| 8 | create_chart | ✅ PASS | 254ms | Validation OK |
| 9 | create_pivot_table | ✅ PASS | 237ms | Validation OK |
| 10 | create_table | ✅ PASS | 262ms | Validation OK |
| 11 | copy_worksheet | ✅ PASS | 263ms | Validation OK |
| 12 | delete_worksheet | ✅ PASS | 265ms | Validation OK |
| 13 | rename_worksheet | ✅ PASS | 235ms | Validation OK |
| 14 | get_workbook_metadata | ✅ PASS | 239ms | Validation OK |
| 15 | merge_cells | ✅ PASS | 249ms | Validation OK |
| 16 | unmerge_cells | ✅ PASS | 246ms | Validation OK |
| 17 | get_merged_cells | ✅ PASS | 247ms | Validation OK |
| 18 | copy_range | ✅ PASS | 247ms | Validation OK |
| 19 | delete_range | ✅ PASS | 238ms | Validation OK |
| 20 | validate_excel_range | ✅ PASS | 257ms | Validation OK |
| 21 | get_data_validation_info | ✅ PASS | 243ms | Validation OK |
| 22 | insert_rows | ✅ PASS | 243ms | Validation OK |
| 23 | insert_columns | ✅ PASS | 258ms | Validation OK |
| 24 | delete_sheet_rows | ✅ PASS | 240ms | Validation OK |
| 25 | delete_sheet_columns | ✅ PASS | 239ms | Validation OK |

**Summary:** 25 passed, 0 failed

---

### gdrive
**Tools:** 4

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | gdrive_search | ❌ FAIL | 252ms | Failed to call tool: tool 'gdrive_search |
| 2 | gdrive_read_file | ❌ FAIL | 240ms | Failed to call tool: tool 'gdrive_read_f |
| 3 | gsheets_update_cell | ❌ FAIL | 257ms | Failed to call tool: tool 'gsheets_updat |
| 4 | gsheets_read | ✅ PASS | 261ms | Validation OK |

**Summary:** 1 passed, 3 failed

---

### github
**Tools:** 17

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | create_or_update_file | ❌ FAIL | 241ms | Failed to call tool: tool 'create_or_upd |
| 2 | search_repositories | ❌ FAIL | 275ms | Failed to call tool: tool 'search_reposi |
| 3 | create_repository | ❌ FAIL | 239ms | Failed to call tool: tool 'create_reposi |
| 4 | get_file_contents | ❌ FAIL | 240ms | Failed to call tool: tool 'get_file_cont |
| 5 | push_files | ❌ FAIL | 249ms | Failed to call tool: tool 'push_files' o |
| 6 | create_issue | ❌ FAIL | 247ms | Failed to call tool: tool 'create_issue' |
| 7 | create_pull_request | ❌ FAIL | 255ms | Failed to call tool: tool 'create_pull_r |
| 8 | fork_repository | ❌ FAIL | 215ms | Failed to call tool: tool 'fork_reposito |
| 9 | create_branch | ❌ FAIL | 288ms | Failed to call tool: tool 'create_branch |
| 10 | list_commits | ❌ FAIL | 293ms | Failed to call tool: tool 'list_commits' |
| 11 | list_issues | ❌ FAIL | 305ms | Failed to call tool: tool 'list_issues'  |
| 12 | update_issue | ❌ FAIL | 275ms | Failed to call tool: tool 'update_issue' |
| 13 | add_issue_comment | ❌ FAIL | 270ms | Failed to call tool: tool 'add_issue_com |
| 14 | search_code | ❌ FAIL | 259ms | Failed to call tool: tool 'search_code'  |
| 15 | search_issues | ❌ FAIL | 265ms | Failed to call tool: tool 'search_issues |
| 16 | search_users | ❌ FAIL | 260ms | Failed to call tool: tool 'search_users' |
| 17 | get_issue | ❌ FAIL | 267ms | Failed to call tool: tool 'get_issue' on |

**Summary:** 0 passed, 17 failed

---

### gitlab
**Tools:** 9

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | create_or_update_file | ❌ FAIL | 238ms | Failed to call tool: tool 'create_or_upd |
| 2 | search_repositories | ❌ FAIL | 271ms | Failed to call tool: tool 'search_reposi |
| 3 | create_repository | ❌ FAIL | 277ms | Failed to call tool: tool 'create_reposi |
| 4 | get_file_contents | ❌ FAIL | 241ms | Failed to call tool: tool 'get_file_cont |
| 5 | push_files | ❌ FAIL | 252ms | Failed to call tool: tool 'push_files' o |
| 6 | create_issue | ❌ FAIL | 254ms | Failed to call tool: tool 'create_issue' |
| 7 | create_merge_request | ❌ FAIL | 220ms | Failed to call tool: tool 'create_merge_ |
| 8 | fork_repository | ❌ FAIL | 227ms | Failed to call tool: tool 'fork_reposito |
| 9 | create_branch | ❌ FAIL | 254ms | Failed to call tool: tool 'create_branch |

**Summary:** 0 passed, 9 failed

---

### gmail
**Tools:** 15

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | categories-list | ❌ FAIL | 15241ms | Timeout/No response |
| 2 | label-add | ❌ FAIL | 229ms | Failed to call tool: server 'gmail' is n |
| 3 | label-delete | ❌ FAIL | 238ms | Failed to call tool: server 'gmail' is n |
| 4 | labels-list | ❌ FAIL | 238ms | Failed to call tool: server 'gmail' is n |
| 5 | message-get | ❌ FAIL | 253ms | Failed to call tool: server 'gmail' is n |
| 6 | message-mark-read | ❌ FAIL | 225ms | Failed to call tool: server 'gmail' is n |
| 7 | message-move-to-trash | ❌ FAIL | 244ms | Failed to call tool: server 'gmail' is n |
| 8 | message-respond | ❌ FAIL | 258ms | Failed to call tool: server 'gmail' is n |
| 9 | message-search | ❌ FAIL | 255ms | Failed to call tool: server 'gmail' is n |
| 10 | message-send | ❌ FAIL | 271ms | Failed to call tool: server 'gmail' is n |
| 11 | messages-export-csv | ❌ FAIL | 286ms | Failed to call tool: server 'gmail' is n |
| 12 | account-me | ❌ FAIL | 262ms | Failed to call tool: server 'gmail' is n |
| 13 | account-switch | ❌ FAIL | 250ms | Failed to call tool: server 'gmail' is n |
| 14 | account-remove | ❌ FAIL | 256ms | Failed to call tool: server 'gmail' is n |
| 15 | account-list | ❌ FAIL | 259ms | Failed to call tool: server 'gmail' is n |

**Summary:** 0 passed, 15 failed

---

### google-calendar
**Tools:** 6

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | list_calendars | ✅ PASS | 955ms | OK |
| 2 | list_calendar_events | ❌ FAIL | 224ms | Failed to call tool: tool 'list_calendar |
| 3 | create_calendar_event | ❌ FAIL | 241ms | Failed to call tool: tool 'create_calend |
| 4 | get_calendar_event | ❌ FAIL | 251ms | Failed to call tool: tool 'get_calendar_ |
| 5 | edit_calendar_event | ❌ FAIL | 266ms | Failed to call tool: tool 'edit_calendar |
| 6 | delete_calendar_event | ❌ FAIL | 262ms | Failed to call tool: tool 'delete_calend |

**Summary:** 1 passed, 5 failed

---

### grep-mcp
**Tools:** 2

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | grep_query | ✅ PASS | 310ms | OK |
| 2 | gitee_query | ✅ PASS | 272ms | OK |

**Summary:** 2 passed, 0 failed

---

### imap-mobilis
**Tools:** 12

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | list_mailboxes | ✅ PASS | 2503ms | OK |
| 2 | create_mailbox | ✅ PASS | 616ms | OK |
| 3 | search_emails | ✅ PASS | 2044ms | OK |
| 4 | get_email | ❌ FAIL | 15323ms | Timeout/No response |
| 5 | move_emails | ❌ FAIL | 316ms | Failed to call tool: server 'imap-mobili |
| 6 | delete_emails | ❌ FAIL | 313ms | Failed to call tool: server 'imap-mobili |
| 7 | mark_seen | ❌ FAIL | 313ms | Failed to call tool: server 'imap-mobili |
| 8 | add_flags | ❌ FAIL | 378ms | Failed to call tool: server 'imap-mobili |
| 9 | remove_flags | ❌ FAIL | 328ms | Failed to call tool: server 'imap-mobili |
| 10 | delete_mailbox | ❌ FAIL | 339ms | Failed to call tool: server 'imap-mobili |
| 11 | list_all_emails | ❌ FAIL | 335ms | Failed to call tool: server 'imap-mobili |
| 12 | send_mail | ❌ FAIL | 374ms | Failed to call tool: server 'imap-mobili |

**Summary:** 3 passed, 9 failed

---

### imap-smsprofi
**Tools:** 12

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | list_mailboxes | ✅ PASS | 2545ms | OK |
| 2 | create_mailbox | ✅ PASS | 700ms | OK |
| 3 | search_emails | ✅ PASS | 2049ms | OK |
| 4 | get_email | ❌ FAIL | 15320ms | Timeout/No response |
| 5 | move_emails | ❌ FAIL | 366ms | Failed to call tool: server 'imap-smspro |
| 6 | delete_emails | ❌ FAIL | 376ms | Failed to call tool: server 'imap-smspro |
| 7 | mark_seen | ❌ FAIL | 342ms | Failed to call tool: server 'imap-smspro |
| 8 | add_flags | ❌ FAIL | 341ms | Failed to call tool: server 'imap-smspro |
| 9 | remove_flags | ❌ FAIL | 488ms | Failed to call tool: server 'imap-smspro |
| 10 | delete_mailbox | ❌ FAIL | 436ms | Failed to call tool: server 'imap-smspro |
| 11 | list_all_emails | ❌ FAIL | 454ms | Failed to call tool: server 'imap-smspro |
| 12 | send_mail | ❌ FAIL | 459ms | Failed to call tool: server 'imap-smspro |

**Summary:** 3 passed, 9 failed

---

### markdownify
**Tools:** 10

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | audio-to-markdown | ✅ PASS | 489ms | Validation OK |
| 2 | bing-search-to-markdown | ✅ PASS | 435ms | Validation OK |
| 3 | docx-to-markdown | ✅ PASS | 410ms | Validation OK |
| 4 | get-markdown-file | ✅ PASS | 393ms | Validation OK |
| 5 | image-to-markdown | ✅ PASS | 418ms | Validation OK |
| 6 | pdf-to-markdown | ✅ PASS | 372ms | Validation OK |
| 7 | pptx-to-markdown | ✅ PASS | 355ms | Validation OK |
| 8 | webpage-to-markdown | ✅ PASS | 376ms | Validation OK |
| 9 | xlsx-to-markdown | ✅ PASS | 418ms | Validation OK |
| 10 | youtube-to-markdown | ✅ PASS | 384ms | Validation OK |

**Summary:** 10 passed, 0 failed

---

### mcp-obsidian
**Status:** ERROR - Could not retrieve tools


### mcp-youtube
**Tools:** 3

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | download_youtube_url | ⚠️ WARN | 3112ms | Error downloading video: Error: [generic |
| 2 | search_youtube_videos | ❌ FAIL | 15414ms | Timeout/No response |
| 3 | get_screenshots | ❌ FAIL | 414ms | Failed to call tool: server 'mcp-youtube |

**Summary:** 0 passed, 2 failed

---

### medium
**Tools:** 5

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | publish-article | ❌ FAIL | 410ms | Failed to call tool: tool 'publish-artic |
| 2 | get-my-articles | ⚠️ WARN | 381ms | Error retrieving articles: page.goto: Ta |
| 3 | get-article-content | ❌ FAIL | 434ms | Failed to call tool: tool 'get-article-c |
| 4 | search-medium | ❌ FAIL | 422ms | Failed to call tool: tool 'search-medium |
| 5 | login-to-medium | ⚠️ WARN | 424ms | Login error: page.goto: Target page, con |

**Summary:** 0 passed, 3 failed

---

### mem0
**Tools:** 2

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | add-memory | ✅ PASS | 1181ms | OK |
| 2 | search-memories | ✅ PASS | 606ms | OK |

**Summary:** 2 passed, 0 failed

---

### ms365
**Tools:** 66

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | login | ✅ PASS | 1977ms | OK |
| 2 | logout | ✅ PASS | 386ms | OK |
| 3 | verify-login | ✅ PASS | 495ms | OK |
| 4 | list-accounts | ✅ PASS | 443ms | OK |
| 5 | select-account | ✅ PASS | 449ms | Validation OK |
| 6 | remove-account | ✅ PASS | 423ms | Validation OK |
| 7 | delete-onedrive-file | ✅ PASS | 424ms | Validation OK |
| 8 | list-folder-files | ✅ PASS | 433ms | Validation OK |
| 9 | download-onedrive-file-content | ✅ PASS | 537ms | Validation OK |
| 10 | upload-file-content | ✅ PASS | 413ms | Validation OK |
| 11 | list-excel-worksheets | ✅ PASS | 407ms | Validation OK |
| 12 | create-excel-chart | ✅ PASS | 441ms | Validation OK |
| 13 | format-excel-range | ✅ PASS | 440ms | Validation OK |
| 14 | sort-excel-range | ✅ PASS | 464ms | Validation OK |
| 15 | get-excel-range | ✅ PASS | 410ms | Validation OK |
| 16 | get-drive-root-item | ✅ PASS | 430ms | Validation OK |
| 17 | get-current-user | ⚠️ WARN | 414ms | {"error":"No valid token found"}  |
| 18 | list-calendars | ⚠️ WARN | 434ms | {"error":"No valid token found"}  |
| 19 | list-specific-calendar-events | ✅ PASS | 429ms | Validation OK |
| 20 | create-specific-calendar-event | ✅ PASS | 413ms | Validation OK |
| 21 | get-specific-calendar-event | ✅ PASS | 396ms | Validation OK |
| 22 | update-specific-calendar-event | ✅ PASS | 474ms | Validation OK |
| 23 | delete-specific-calendar-event | ✅ PASS | 399ms | Validation OK |
| 24 | get-calendar-view | ✅ PASS | 392ms | Validation OK |
| 25 | list-outlook-contacts | ⚠️ WARN | 384ms | {"error":"No valid token found"}  |
| 26 | create-outlook-contact | ✅ PASS | 373ms | Validation OK |
| 27 | get-outlook-contact | ✅ PASS | 427ms | Validation OK |
| 28 | update-outlook-contact | ✅ PASS | 346ms | Validation OK |
| 29 | delete-outlook-contact | ✅ PASS | 328ms | Validation OK |
| 30 | list-drives | ⚠️ WARN | 346ms | {"error":"No valid token found"}  |
| 31 | list-calendar-events | ⚠️ WARN | 326ms | {"error":"No valid token found"}  |
| 32 | create-calendar-event | ✅ PASS | 361ms | Validation OK |
| 33 | get-calendar-event | ✅ PASS | 399ms | Validation OK |
| 34 | update-calendar-event | ✅ PASS | 466ms | Validation OK |
| 35 | delete-calendar-event | ✅ PASS | 354ms | Validation OK |
| 36 | list-mail-folders | ⚠️ WARN | 349ms | {"error":"No valid token found"}  |
| 37 | list-mail-folder-messages | ✅ PASS | 362ms | Validation OK |
| 38 | list-mail-messages | ⚠️ WARN | 395ms | {"error":"No valid token found"}  |
| 39 | create-draft-email | ✅ PASS | 430ms | Validation OK |
| 40 | get-mail-message | ✅ PASS | 354ms | Validation OK |
| 41 | delete-mail-message | ✅ PASS | 425ms | Validation OK |
| 42 | list-mail-attachments | ✅ PASS | 460ms | Validation OK |
| 43 | add-mail-attachment | ✅ PASS | 420ms | Validation OK |
| 44 | get-mail-attachment | ✅ PASS | 424ms | Validation OK |
| 45 | delete-mail-attachment | ✅ PASS | 387ms | Validation OK |
| 46 | move-mail-message | ✅ PASS | 365ms | Validation OK |
| 47 | list-onenote-notebooks | ⚠️ WARN | 485ms | {"error":"No valid token found"}  |
| 48 | list-onenote-notebook-sections | ✅ PASS | 410ms | Validation OK |
| 49 | create-onenote-page | ✅ PASS | 421ms | Validation OK |
| 50 | get-onenote-page-content | ✅ PASS | 445ms | Validation OK |
| 51 | list-onenote-section-pages | ✅ PASS | 411ms | Validation OK |
| 52 | list-planner-tasks | ⚠️ WARN | 458ms | {"error":"No valid token found"}  |
| 53 | send-mail | ✅ PASS | 416ms | Validation OK |
| 54 | list-todo-task-lists | ⚠️ WARN | 377ms | {"error":"No valid token found"}  |
| 55 | list-todo-tasks | ✅ PASS | 408ms | Validation OK |
| 56 | create-todo-task | ✅ PASS | 343ms | Validation OK |
| 57 | get-todo-task | ✅ PASS | 385ms | Validation OK |
| 58 | update-todo-task | ✅ PASS | 404ms | Validation OK |
| 59 | delete-todo-task | ✅ PASS | 380ms | Validation OK |
| 60 | get-planner-plan | ✅ PASS | 376ms | Validation OK |
| 61 | list-plan-tasks | ✅ PASS | 405ms | Validation OK |
| 62 | create-planner-task | ✅ PASS | 412ms | Validation OK |
| 63 | get-planner-task | ✅ PASS | 452ms | Validation OK |
| 64 | update-planner-task | ✅ PASS | 431ms | Validation OK |
| 65 | update-planner-task-details | ✅ PASS | 473ms | Validation OK |
| 66 | search-query | ✅ PASS | 429ms | Validation OK |

**Summary:** 56 passed, 0 failed

---

### ms365-private
**Tools:** 66

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | login | ✅ PASS | 1918ms | OK |
| 2 | logout | ✅ PASS | 413ms | OK |
| 3 | verify-login | ✅ PASS | 435ms | OK |
| 4 | list-accounts | ✅ PASS | 427ms | OK |
| 5 | select-account | ✅ PASS | 463ms | Validation OK |
| 6 | remove-account | ✅ PASS | 453ms | Validation OK |
| 7 | delete-onedrive-file | ✅ PASS | 444ms | Validation OK |
| 8 | list-folder-files | ✅ PASS | 435ms | Validation OK |
| 9 | download-onedrive-file-content | ✅ PASS | 514ms | Validation OK |
| 10 | upload-file-content | ✅ PASS | 457ms | Validation OK |
| 11 | list-excel-worksheets | ✅ PASS | 402ms | Validation OK |
| 12 | create-excel-chart | ✅ PASS | 384ms | Validation OK |
| 13 | format-excel-range | ✅ PASS | 401ms | Validation OK |
| 14 | sort-excel-range | ✅ PASS | 392ms | Validation OK |
| 15 | get-excel-range | ✅ PASS | 408ms | Validation OK |
| 16 | get-drive-root-item | ✅ PASS | 460ms | Validation OK |
| 17 | get-current-user | ⚠️ WARN | 400ms | {"error":"No valid token found"}  |
| 18 | list-calendars | ⚠️ WARN | 406ms | {"error":"No valid token found"}  |
| 19 | list-specific-calendar-events | ✅ PASS | 384ms | Validation OK |
| 20 | create-specific-calendar-event | ✅ PASS | 405ms | Validation OK |
| 21 | get-specific-calendar-event | ✅ PASS | 384ms | Validation OK |
| 22 | update-specific-calendar-event | ✅ PASS | 395ms | Validation OK |
| 23 | delete-specific-calendar-event | ✅ PASS | 469ms | Validation OK |
| 24 | get-calendar-view | ✅ PASS | 458ms | Validation OK |
| 25 | list-outlook-contacts | ⚠️ WARN | 434ms | {"error":"No valid token found"}  |
| 26 | create-outlook-contact | ✅ PASS | 419ms | Validation OK |
| 27 | get-outlook-contact | ✅ PASS | 399ms | Validation OK |
| 28 | update-outlook-contact | ✅ PASS | 420ms | Validation OK |
| 29 | delete-outlook-contact | ✅ PASS | 438ms | Validation OK |
| 30 | list-drives | ⚠️ WARN | 421ms | {"error":"No valid token found"}  |
| 31 | list-calendar-events | ⚠️ WARN | 374ms | {"error":"No valid token found"}  |
| 32 | create-calendar-event | ✅ PASS | 438ms | Validation OK |
| 33 | get-calendar-event | ✅ PASS | 424ms | Validation OK |
| 34 | update-calendar-event | ✅ PASS | 469ms | Validation OK |
| 35 | delete-calendar-event | ✅ PASS | 521ms | Validation OK |
| 36 | list-mail-folders | ⚠️ WARN | 442ms | {"error":"No valid token found"}  |
| 37 | list-mail-folder-messages | ✅ PASS | 454ms | Validation OK |
| 38 | list-mail-messages | ⚠️ WARN | 449ms | {"error":"No valid token found"}  |
| 39 | create-draft-email | ✅ PASS | 558ms | Validation OK |
| 40 | get-mail-message | ✅ PASS | 489ms | Validation OK |
| 41 | delete-mail-message | ✅ PASS | 446ms | Validation OK |
| 42 | list-mail-attachments | ✅ PASS | 428ms | Validation OK |
| 43 | add-mail-attachment | ✅ PASS | 475ms | Validation OK |
| 44 | get-mail-attachment | ✅ PASS | 438ms | Validation OK |
| 45 | delete-mail-attachment | ✅ PASS | 429ms | Validation OK |
| 46 | move-mail-message | ✅ PASS | 569ms | Validation OK |
| 47 | list-onenote-notebooks | ⚠️ WARN | 446ms | {"error":"No valid token found"}  |
| 48 | list-onenote-notebook-sections | ✅ PASS | 463ms | Validation OK |
| 49 | create-onenote-page | ✅ PASS | 431ms | Validation OK |
| 50 | get-onenote-page-content | ✅ PASS | 411ms | Validation OK |
| 51 | list-onenote-section-pages | ✅ PASS | 460ms | Validation OK |
| 52 | list-planner-tasks | ⚠️ WARN | 484ms | {"error":"No valid token found"}  |
| 53 | send-mail | ✅ PASS | 427ms | Validation OK |
| 54 | list-todo-task-lists | ⚠️ WARN | 430ms | {"error":"No valid token found"}  |
| 55 | list-todo-tasks | ✅ PASS | 412ms | Validation OK |
| 56 | create-todo-task | ✅ PASS | 426ms | Validation OK |
| 57 | get-todo-task | ✅ PASS | 415ms | Validation OK |
| 58 | update-todo-task | ✅ PASS | 371ms | Validation OK |
| 59 | delete-todo-task | ✅ PASS | 436ms | Validation OK |
| 60 | get-planner-plan | ✅ PASS | 435ms | Validation OK |
| 61 | list-plan-tasks | ✅ PASS | 434ms | Validation OK |
| 62 | create-planner-task | ✅ PASS | 428ms | Validation OK |
| 63 | get-planner-task | ✅ PASS | 481ms | Validation OK |
| 64 | update-planner-task | ✅ PASS | 432ms | Validation OK |
| 65 | update-planner-task-details | ✅ PASS | 444ms | Validation OK |
| 66 | search-query | ✅ PASS | 434ms | Validation OK |

**Summary:** 56 passed, 0 failed

---

### multi-ai-mcp
**Tools:** 17

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | server_status | ✅ PASS | 431ms | OK |
| 2 | ask_gemini | ✅ PASS | 400ms | OK |
| 3 | gemini_code_review | ✅ PASS | 3160ms | OK |
| 4 | gemini_think_deep | ✅ PASS | 563ms | OK |
| 5 | gemini_brainstorm | ✅ PASS | 610ms | OK |
| 6 | gemini_debug | ✅ PASS | 621ms | OK |
| 7 | gemini_architecture | ✅ PASS | 564ms | OK |
| 8 | ask_openai | ✅ PASS | 2712ms | OK |
| 9 | openai_code_review | ✅ PASS | 4560ms | OK |
| 10 | openai_think_deep | ❌ FAIL | 15388ms | Timeout/No response |
| 11 | openai_brainstorm | ❌ FAIL | 424ms | Failed to call tool: server 'multi-ai-mc |
| 12 | openai_debug | ❌ FAIL | 371ms | Failed to call tool: server 'multi-ai-mc |
| 13 | openai_architecture | ❌ FAIL | 418ms | Failed to call tool: server 'multi-ai-mc |
| 14 | ask_all_ais | ❌ FAIL | 620ms | Failed to call tool: server 'multi-ai-mc |
| 15 | ai_debate | ❌ FAIL | 447ms | Failed to call tool: server 'multi-ai-mc |
| 16 | collaborative_solve | ❌ FAIL | 401ms | Failed to call tool: server 'multi-ai-mc |
| 17 | ai_consensus | ❌ FAIL | 398ms | Failed to call tool: server 'multi-ai-mc |

**Summary:** 9 passed, 8 failed

---

### mysql
**Status:** ERROR - Could not retrieve tools


### mysql-prod
**Status:** ERROR - Could not retrieve tools


### n8n
**Status:** ERROR - Could not retrieve tools


### neo4j
**Tools:** 4

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | memory_store | ✅ PASS | 468ms | Validation OK |
| 2 | memory_find | ✅ PASS | 397ms | Validation OK |
| 3 | memory_modify | ✅ PASS | 382ms | Validation OK |
| 4 | database_switch | ✅ PASS | 464ms | Validation OK |

**Summary:** 4 passed, 0 failed

---

### notebooklm
**Tools:** 31

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | notebook_list | ✅ PASS | 443ms | OK |
| 2 | notebook_create | ✅ PASS | 440ms | OK |
| 3 | notebook_get | ✅ PASS | 464ms | Validation OK |
| 4 | notebook_describe | ✅ PASS | 479ms | Validation OK |
| 5 | source_describe | ✅ PASS | 423ms | Validation OK |
| 6 | notebook_add_url | ✅ PASS | 434ms | Validation OK |
| 7 | notebook_add_text | ✅ PASS | 439ms | Validation OK |
| 8 | notebook_add_drive | ✅ PASS | 426ms | Validation OK |
| 9 | notebook_query | ✅ PASS | 425ms | Validation OK |
| 10 | notebook_delete | ✅ PASS | 430ms | Validation OK |
| 11 | notebook_rename | ✅ PASS | 405ms | Validation OK |
| 12 | chat_configure | ✅ PASS | 441ms | Validation OK |
| 13 | source_list_drive | ✅ PASS | 393ms | Validation OK |
| 14 | source_sync_drive | ✅ PASS | 441ms | Validation OK |
| 15 | source_delete | ✅ PASS | 428ms | Validation OK |
| 16 | research_start | ✅ PASS | 426ms | Validation OK |
| 17 | research_status | ✅ PASS | 410ms | Validation OK |
| 18 | research_import | ✅ PASS | 412ms | Validation OK |
| 19 | audio_overview_create | ✅ PASS | 400ms | Validation OK |
| 20 | video_overview_create | ✅ PASS | 429ms | Validation OK |
| 21 | studio_status | ✅ PASS | 402ms | Validation OK |
| 22 | studio_delete | ✅ PASS | 464ms | Validation OK |
| 23 | infographic_create | ✅ PASS | 391ms | Validation OK |
| 24 | slide_deck_create | ✅ PASS | 425ms | Validation OK |
| 25 | report_create | ✅ PASS | 452ms | Validation OK |
| 26 | flashcards_create | ✅ PASS | 469ms | Validation OK |
| 27 | quiz_create | ✅ PASS | 417ms | Validation OK |
| 28 | data_table_create | ✅ PASS | 385ms | Validation OK |
| 29 | mind_map_create | ✅ PASS | 406ms | Validation OK |
| 30 | mind_map_list | ✅ PASS | 410ms | Validation OK |
| 31 | save_auth_tokens | ✅ PASS | 438ms | Validation OK |

**Summary:** 31 passed, 0 failed

---

### pal-mcp
**Status:** ERROR - Could not retrieve tools


### playwright
**Status:** ERROR - Could not retrieve tools


### postgres
**Tools:** 1

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | query | ❌ FAIL | 646ms | Failed to call tool: tool 'query' on ser |

**Summary:** 0 passed, 1 failed

---

### postman
**Status:** ERROR - Could not retrieve tools


### qdrant
**Tools:** 4

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|
| 1 | list_collections | ✅ PASS | 532ms | OK |
| 2 | add_documents | ❌ FAIL | 2649ms | Failed to call tool: tool 'add_documents |
| 3 | search | ❌ FAIL | 767ms | Failed to call tool: server 'qdrant' is  |
| 4 | delete_collection | ❌ FAIL | 627ms | Failed to call tool: server 'qdrant' is  |

**Summary:** 1 passed, 3 failed

---

### reddit
**Tools:** 0

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|

**Summary:** 0 passed, 0 failed

---

### supabase
**Tools:** 0

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|

**Summary:** 0 passed, 0 failed

---

### targetprocess
**Tools:** 0

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|

**Summary:** 0 passed, 0 failed

---

### taskmaster
**Tools:** 0

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|

**Summary:** 0 passed, 0 failed

---

### test-filesystem-server
**Tools:** 0

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|

**Summary:** 0 passed, 0 failed

---

### zen-mcp-server
**Tools:** 0

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|

**Summary:** 0 passed, 0 failed

---

### zep-graphiti
**Tools:** 0

| # | Tool | Status | Response Time | Details |
|---|------|--------|---------------|---------|

**Summary:** 0 passed, 0 failed

---
