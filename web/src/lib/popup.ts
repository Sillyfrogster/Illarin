/** popupMotionClass grows a dropdown, select or combobox popup from the side it opened on. */
export const popupMotionClass =
  "origin-top [--popup-enter-y:-4px] " +
  "data-[side=top]:origin-bottom data-[side=top]:[--popup-enter-y:4px] " +
  "[[data-side=top]_&]:origin-bottom [[data-side=top]_&]:[--popup-enter-y:4px] " +
  "data-[side=left]:origin-right data-[side=left]:[--popup-enter-y:0px] " +
  "[[data-side=left]_&]:origin-right [[data-side=left]_&]:[--popup-enter-y:0px] " +
  "data-[side=right]:origin-left data-[side=right]:[--popup-enter-y:0px] " +
  "[[data-side=right]_&]:origin-left [[data-side=right]_&]:[--popup-enter-y:0px]";
