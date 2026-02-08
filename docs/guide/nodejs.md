# Node.js Integration

Forge can automatically sync the version in your `package.json` file when creating git tags. This is useful for projects that use both Go and Node.js, or Go projects that maintain a `package.json` for frontend tooling.

## How It Works

When Node.js integration is enabled, the `forge bump` command will:

1. Calculate the next version
2. Update the `version` field in `package.json`
3. Stage the change with `git add`
4. Commit the change
5. Create the git tag on the new commit (including the `package.json` update)

This ensures the git tag always points to a commit where `package.json` has the correct version.

## Configuration

Enable Node.js integration in your `forge.yaml`:

```yaml
nodejs:
  enabled: true
  package_path: ""  # defaults to ./package.json
```

### Custom Path

For monorepos or projects where `package.json` is not in the root:

```yaml
nodejs:
  enabled: true
  package_path: "frontend/package.json"
```

## Example

Before bump:
```json
{
  "name": "my-app",
  "version": "1.2.3"
}
```

After `forge bump --bump minor`:
```json
{
  "name": "my-app",
  "version": "1.3.0"
}
```

The version in `package.json` is updated **without the prefix** — it uses the clean version string (e.g., `1.3.0`, not `v1.3.0`), following npm conventions.

## Dry Run

With `--dry-run`, the `package.json` is not modified:

```bash
forge bump --bump minor --dry-run
```
