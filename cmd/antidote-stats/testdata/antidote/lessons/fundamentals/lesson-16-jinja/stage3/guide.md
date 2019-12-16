## Using Jinja for Configuration Templates

**Contributed by: [@ShrutiVPawaskar](https://github.com/ShrutiVPawaskar) and [@shahbhoomi](https://github.com/shahbhoomi)**

---

## Part 3 - "If" and "Set" Statements

Now that you have tried the `for` loop, lets up-level. In this part we will try `if` and `set` statement along with `for` loops.

Again, we want to start the Python interpreter and import `Environment` module from the Jinja2 library:

```
python
from jinja2 import Environment
```
<button type="button" class="btn btn-primary btn-sm" onclick="runSnippetInTab('linux1', this)">Run this snippet</button>

Here we are redefining `interface`, the list of dictionaries we defined in part 2.

```
interfaces = [{'interface': 'ge-0/0/0', 'ip_address': '192.168.1.1'},
              {'interface': 'ge-0/0/1', 'ip_address': '10.10.1.1'},
              {'interface': 'fxp0', 'ip_address': '172.16.1.1'}]

```
<button type="button" class="btn btn-primary btn-sm" onclick="runSnippetInTab('linux1', this)">Run this snippet</button>

In Part2, we generated configurations for all interfaces that were passed to the template. Wwhat if you are only interested in generating the  configuration for the management interface? For this particular example we will be using the `if` condition. It is similar to the python `if` condition, except for some slight syntax differences:

```
{% if condition %}
 ...
{% endif %}
```

In the following snippet we have used the `for` loop like in previous stages, but instead of directly printing out all the interfaces here we are checking if the interface value matches our management interface `fxp0`. If it does match, we substitute its value and the value of the corresponding ip address in the template using `render()`.

```
env = Environment(trim_blocks=True, lstrip_blocks=True)
ipaddr_template = env.from_string('''
{% for item in interfaces %}
{% if item.interface == 'fxp0' %}
interfaces {
    {{ item.interface }} {
        unit 0 {
            family inet {
                address {{item.ip_address}};
            }
        }
    }
}
{% endif %}
{% endfor %}''')

render_1 = ipaddr_template.render(interfaces=interfaces)
print(str(render_1))

```
<button type="button" class="btn btn-primary btn-sm" onclick="runSnippetInTab('linux1', this)">Run this snippet</button>


The output shows that it just printed the entry of the management interface. Thus we learnt the power of `if` statements to filter out our data based on certain conditions.

Now, what if we decide to change the management interface to `ge-0/0/0`. One approach is to change the `fxp0` to `ge-0/0/0` in the `if` statement.
A better approach is to define a variable called `mgmt_interface` and, change the value of this variable when we decide to change the management interface.

Jinja uses the `set` statement to define variables which can be used within the template. The syntax is defined below:

```
{% set variable_name: variable_value %}
```

In the snippet below:
`set_template = '''{% set mgmt_interface = 'ge-0/0/0' %}` declares a variable `mgmt_interface` and sets its value to `ge-0/0/0`.
`{% if item.interface == mgmt_interface %}` checks if the item in the for loop has a key interface whose value is same as the value that we set for `mgmt_interface`.

If the value is same, it loads the template with the corresponding data, if not, it skips that element of list `interfaces` and moves on to the next.

```
set_template = '''{% set mgmt_interface = 'ge-0/0/0' %}
{% for item in interfaces %}
{% if item.interface == mgmt_interface %}
interfaces {
    {{ item.interface }} {
        unit 0 {
            family inet {
                address {{item.ip_address}};
            }
        }
    }
}
{% endif %}
{% endfor %}'''

int_template = env.from_string(set_template)
render_2 = int_template.render(interfaces=interfaces)

print(str(render_2))
```
<button type="button" class="btn btn-primary btn-sm" onclick="runSnippetInTab('linux1', this)">Run this snippet</button>

In the next stage, we'll dive deeper into using YAML files for defining variables for Jinja2 templates.
