import { expect, test } from "@playwright/test";
import {
  createGroup,
  createTemplate,
  requiresEnterpriseLicense,
  updateTemplateSettings,
} from "../helpers";

test("template update with new name redirects on successful submit", async ({
  page,
}) => {
  const templateName = await createTemplate(page);

  await updateTemplateSettings(page, templateName, {
    name: "new-name",
  });
});

test("add and remove a group", async ({ page }) => {
  requiresEnterpriseLicense();

  const templateName = await createTemplate(page);
  const groupName = await createGroup(page);

  await page.goto(`/templates/${templateName}/settings/permissions`, {
    waitUntil: "domcontentloaded",
  });
  await expect(page).toHaveURL(
    `/templates/${templateName}/settings/permissions`,
  );

  // Type the first half of the group name
  await page
    .getByPlaceholder("Search for user or group", { exact: true })
    .fill(groupName.slice(0, 4));

  // Select the group from the list and add it
  await page.getByText(groupName).click();
  await page.getByText("Add member").click();
  await expect(
    page.locator(".MuiTable-root").getByText(groupName),
  ).toBeVisible();
});

test("require latest version", async ({ page }) => {
  requiresEnterpriseLicense();

  const templateName = await createTemplate(page);

  await page.goto(`/templates/${templateName}/settings`, {
    waitUntil: "domcontentloaded",
  });
  await expect(page).toHaveURL(`/templates/${templateName}/settings`);
  let checkbox = await page.waitForSelector("#require_active_version");
  await checkbox.click();
  await page.getByTestId("form-submit").click();

  await page.goto(`/templates/${templateName}/settings`, {
    waitUntil: "domcontentloaded",
  });
  checkbox = await page.waitForSelector("#require_active_version");
  await checkbox.scrollIntoViewIfNeeded();
  expect(await checkbox.isChecked()).toBe(true);
});
