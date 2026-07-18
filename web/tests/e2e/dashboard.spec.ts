import { expect, test } from "@playwright/test";

test("opens authentication and dashboard preview", async ({ page }) => {
  await page.route("**/api/v1/auth/me", (route) =>
    route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({
        title: "Unauthorized",
        status: 401,
        detail: "Authentication required",
      }),
    }),
  );

  await page.goto("/");
  await expect(page.getByText("Sign in to manage your tunnels.")).toBeVisible();

  await page.getByRole("button", { name: /preview the dashboard/i }).click();
  await expect(page.getByText("troy@example.com")).toBeVisible();
  await expect(page.getByText("checkout.opts.ink")).toBeVisible();
});

test("shows device approval context", async ({ page }) => {
  await page.route("**/api/v1/auth/me", (route) =>
    route.fulfill({ status: 401, body: "{}" }),
  );

  await page.goto("/device/?user_code=ABCD-EFGH");
  await expect(page.getByText(/approve cli sign-in/i)).toBeVisible();
  await expect(page.getByText("ABCD-EFGH")).toBeVisible();
});
