using System.Collections.Generic;
using StardewValley;
using StardewValley.Buffs;
using StardewValley.Menus;

namespace Fixture
{
    public class Mod
    {
        private int own;

        public int Own { get; set; }

        public void SwitchTool(Farmer farmer, int kind)
        {
            switch (kind)
            {
                case 0: farmer.CurrentToolIndex = 1; break;
                case 1: farmer.CurrentToolIndex = 2; break;
                case 2: farmer.CurrentToolIndex = 4; break;
                default: this.own = kind; break;
            }
            this.Own = kind;
        }

        public void Heal(Farmer farmer)
        {
            farmer.health.Value = farmer.maxHealth.Value;
            farmer.NetMoney.Value += 10;
            farmer.stamina = 270;
            Game1.flashAlpha = 1f;
        }

        public void Pockets(Farmer.Pocket pocket) => pocket.Count = 3;

        public BuffEffects Effects() => new BuffEffects { Speed = 2, Defense = { Value = 3 } };

        public ClickableComponent Component(List<ClickableComponent> into)
        {
            var built = new ClickableTextureComponent();
            built.myID = 1;
            into.Add(built);
            built.leftNeighborID = 2;
            return built;
        }

        public void Neighbours(bool current)
        {
            Game1.CurrentComponent.upNeighborID = 3;
            var either = new ClickableComponent();
            if (current)
            {
                either = Game1.CurrentComponent;
            }
            either.rightNeighborID = 4;
        }

        public int Read(Farmer farmer) => farmer.CurrentToolIndex + farmer.maxHealth.Value + this.own;
    }
}

namespace Fixture.Plugins
{
    [BepInEx.BepInPlugin("com.fixture.plugin", "Fixture Plugin", "1.2.3")]
    [BepInEx.BepInDependency("com.fixture.hard")]
    [BepInEx.BepInDependency("com.fixture.min", "2.1.0")]
    [BepInEx.BepInDependency("com.fixture.soft", BepInEx.DependencyFlags.SoftDependency)]
    [BepInEx.BepInIncompatibility("com.fixture.clash")]
    public class Plugin
    {
    }
}

namespace Fixture.Patches
{
    using System.Collections.Generic;
    using HarmonyLib;
    using StardewValley;

    [HarmonyPatch(typeof(Farmer))]
    public static class FarmerPatches
    {
        [HarmonyPatch("Update")]
        [HarmonyPrefix]
        public static bool SkipUpdate() => false;

        [HarmonyPatch(nameof(Farmer.CurrentToolIndex), MethodType.Getter)]
        public static void Postfix()
        {
        }
    }

    [HarmonyPatch(typeof(Game1), "Draw", typeof(int))]
    public static class DrawPatch
    {
        public static IEnumerable<object> Transpiler(IEnumerable<object> code) => code;

        public static void Prefix()
        {
        }
    }

    [HarmonyPatch("StardewValley.Menus.ClickableComponent", "snap")]
    public static class ByName
    {
        [HarmonyFinalizer]
        public static void Done()
        {
        }
    }

    [HarmonyPatch(typeof(Farmer), MethodType.Constructor)]
    public static class Built
    {
        public static void Postfix()
        {
        }
    }

    public static class NotAPatch
    {
        public static void Prefix()
        {
        }
    }
}

namespace Fixture.Hooks
{
    public static class Hooks
    {
        public static void Apply()
        {
            On.StardewValley.Farmer.Update += (orig, self) => orig(self);
            IL.StardewValley.Game1.Update += il => { };
        }
    }
}

namespace Fixture.Settings
{
    using BepInEx.Configuration;
    using CSync.Extensions;
    using ConfigSync;

    public enum Mode
    {
        Calm,
        Wild = 4,
    }

    // Each way a plugin binds a setting: plain, with a description object and a range, shared through ConfigSync at
    // each level, through CSync, and with a default that is not a constant.
    public class Binds
    {
        public void Bind(ConfigFile config, SyncedConfig shared, float notConstant)
        {
            config.Bind("General", "Enabled", true, "Turns it on.");
            config.Bind("General", "Name", "Fixture", "A name.");
            config.Bind("Numbers", "Count", 12, new ConfigDescription("How many.", new AcceptableValueRange<int>(1, 100)));
            config.Bind("Numbers", "Speed", 1.5f, new ConfigDescription("How fast.", new AcceptableValueRange<float>(0.5f, 10f)));
            config.Bind("Numbers", "Big", 3000000000L, "A long.");
            config.Bind("Numbers", "Ratio", 0.25, "A double.");
            config.Bind("Numbers", "Below", -3, "Negative.");
            shared.Share(config.Bind("Shared", "Mode", Mode.Wild, "Picked by the host."));
            shared.Share(config.Bind("Shared", "Hint", false, "The host suggests it."), Sync.HostRecommended);
            config.BindSyncedEntry("Shared", "Through CSync", 7, "Synced by CSync.");
            config.Bind("General", "Unknown", notConstant, "Not a constant.");
        }
    }
}
