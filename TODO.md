**TASK #1**
# MAIN TASK
On the client side, when the user watches a product, from all the settings that already appear like size, type of clothe, color,
quantity, etc. It should also appear the next two things.
- Icon display: This means if the user wants only the icon to be displayed or also the name of that icon, example: if the user is
watching a python shirt, it should appear the option if the user only wants the icon to be displayed or if it also wants the name of
that icon, if user selected the second option, the order should be the icon on the left and the name next to the icon on the right,
vertically aligned to the icon.
- Icon positions: Only the first option can be done in the front, from now on, all the clothes can only have design on the front, 
delete all the settings and make UI/UX fixes to let the user put something on the back, only the front of all the clothes can be
designed.

# WHAT TO-DO
- To all the clothing, only make it black or white, let the user decide which color, only those two.
- The icons or icon with name, only have to follow certain designs, the designs are in /public/clothing, the images show where
these should be positioned, they should be positioned from 4 to 8 centimeters down of the neck, can be positioned on the right or 
left at the same height all the positions.

# WHAT NOT TO-DO
- Do not add unnecesary lines of code, make the reviews, check if the code can be factorized (only new code), do not check the entire
codebase following this rule.
- Do not delete neccesary code from the codebase, when deleting code, check if there is no errors, if there are errors, re-check
the code that has been deleted to see what was wrong.

# FEEDBACK
If you have any questions, feel free to put them on the terminal so i can answer all of them.

**TASK #2**
# ORCHESTRATION
## Multi-Agent Workflow

You are the lead/orchestrator.

You have two Codex workers available.

### Codex 1
Role: implementation

Send tasks using:

./scripts/1send.codex.sh "TASK"

Read its output using:

./scripts/1read.codex.sh

### Codex 2
Role: testing and code review

Send tasks using:

./scripts/2send.codex.sh "TASK"

Read its output using:

./scripts/2read.codex.sh

### Workflow

For every feature:

1. Analyze the request.
2. Create an implementation plan.
3. Send implementation work to Codex 1.
4. Monitor Codex 1.
5. When implementation is complete, send review/testing to Codex 2.
6. Monitor Codex 2.
7. If Codex 2 finds problems, send fixes to Codex 1.
8. Repeat until tests pass.
9. Perform your own final review.
10. Report completion to the user.

Do not implement the feature yourself unless necessary.
Act primarily as the lead engineer.

# MAIN TASK
Complete full aplication to be ready for deployment, create a docker image with a database in postgresql with supabase, to test 
everything, save database configuration on the /backend/database folder, inside this folder there are 3 .sql files:
- scheme: here you are going to save all the database structure, scheme, rls, grants, all of that
- exec: here you are going to save seed data that needs to be on the database for testing and deployment
- alter: here you are going to save modifications that the database needs every feature we integrate
Also test that everything is responsive and check out payments section, i said that i was going to integrate crypto payments, but
for now, only stripe payments will be accepted, so delete any other type of payments

# WHAT TO DO
- Check that the store is ready for production
- Check payments section
- Create all merge and save it on a .sql file so i can insert it on a database production then
- Create all merge for every logo and color, for example for every logo, create the cloth with every position the logo can be in it,
the logo if i remember well can be in the center, right and left in the top o the clothing, so generate all the images and put them 
on a folder, reference that folder to the database so the api can send the images right with its corresponding data

# WHAT NOT TO DO
- Do not write code that it is not necessary
- Do not write or break internal code that breaks the bussiness logic
- Do not overwrite code that it is necessary
- Do not break internal code

# FEEDBACK
If you have any questions feel free to ask via claude code terminal
