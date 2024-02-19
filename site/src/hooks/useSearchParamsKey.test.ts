import { useSearchParamKey } from "./useSearchParamsKey";
import { renderHookWithAuth2 } from "testHelpers/renderHelpers";
import { act, waitFor } from "@testing-library/react";

/**
 * Tried to extract the setup logic into one place, but it got surprisingly
 * messy. Went with straightforward approach of calling things individually
 */
describe(useSearchParamKey.name, () => {
  it("Returns out a default value of an empty string if the key does not exist in URL", async () => {
    const { result } = await renderHookWithAuth2(
      () => useSearchParamKey("blah"),
      { routingOptions: { route: `/` } },
    );

    expect(result.current.value).toEqual("");
  });

  it("Uses the 'defaultValue' config override if provided", async () => {
    const defaultValue = "dogs";
    const { result } = await renderHookWithAuth2(
      () => useSearchParamKey("blah", { defaultValue }),
      { routingOptions: { route: `/` } },
    );

    expect(result.current.value).toEqual(defaultValue);
  });

  it("Is able to read to read keys from the URL on mounting render", async () => {
    const key = "blah";
    const value = "cats";

    const { result } = await renderHookWithAuth2(() => useSearchParamKey(key), {
      routingOptions: {
        route: `/?${key}=${value}`,
      },
    });

    expect(result.current.value).toEqual(value);
  });

  it("Updates state and URL when the setValue callback is called with a new value", async () => {
    const key = "blah";
    const initialValue = "cats";

    const { result, getSearchParamsSnapshot } = await renderHookWithAuth2(
      () => useSearchParamKey(key),
      {
        routingOptions: {
          route: `/?${key}=${initialValue}`,
        },
      },
    );

    const newValue = "dogs";
    act(() => result.current.onValueChange(newValue));
    await waitFor(() => expect(result.current.value).toEqual(newValue));

    const params = getSearchParamsSnapshot();
    expect(params.get(key)).toEqual(newValue);
  });

  it("Clears value for the given key from the state and URL when removeValue is called", async () => {
    const key = "blah";
    const initialValue = "cats";

    const { result, getSearchParamsSnapshot } = await renderHookWithAuth2(
      () => useSearchParamKey(key),
      {
        routingOptions: {
          route: `/?${key}=${initialValue}`,
        },
      },
    );

    act(() => result.current.removeValue());
    await waitFor(() => expect(result.current.value).toEqual(""));

    const params = getSearchParamsSnapshot();
    expect(params.get(key)).toEqual(null);
  });

  it("Does not have methods change previous values if 'key' argument changes during re-renders", async () => {
    const readonlyKey = "readonlyKey";
    const mutableKey = "mutableKey";
    const initialReadonlyValue = "readonly";
    const initialMutableValue = "mutable";

    const { result, rerender, getSearchParamsSnapshot } =
      await renderHookWithAuth2(({ key }) => useSearchParamKey(key), {
        routingOptions: {
          route: `/?${readonlyKey}=${initialReadonlyValue}&${mutableKey}=${initialMutableValue}`,
        },

        renderOptions: {
          initialProps: { key: readonlyKey },
        },
      });

    const swapValue = "dogs";
    rerender({ key: mutableKey });
    act(() => result.current.onValueChange(swapValue));
    await waitFor(() => expect(result.current.value).toEqual(swapValue));

    const params = getSearchParamsSnapshot();
    expect(params.get(readonlyKey)).toEqual(initialReadonlyValue);
    expect(params.get(mutableKey)).toEqual(swapValue);
  });

  it.skip("Does not update the history stack for any of its methods if 'replace' config option is true", () => {
    expect.hasAssertions();
  });
});
