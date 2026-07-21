package recall_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/recall"
)

// Fusion is pure arithmetic over ranks. These tests take no database, and if
// one ever needs one, the ranking has stopped being a function of the ranks.

func ranked(ids ...string) []recall.Ranked {
	out := make([]recall.Ranked, len(ids))
	for i, id := range ids {
		out[i] = recall.Ranked{ID: id, Rank: i + 1, Raw: float64(len(ids) - i)}
	}
	return out
}

func TestRankIsOneBased(t *testing.T) {
	got := recall.FuseRRF(map[recall.Arm][]recall.Ranked{
		recall.ArmDense: {{ID: "a", Rank: 1}},
	})
	require.Len(t, got, 1)
	// 1/(60+1), not 1/(60+0). A 0-based rank shifts every score by ~1.6% —
	// enough to reorder near-ties, never enough to notice.
	require.InDelta(t, 1.0/61.0, got[0].Score, 1e-12)
}

func TestRankZeroIsRejectedRatherThanScored(t *testing.T) {
	got := recall.FuseRRF(map[recall.Arm][]recall.Ranked{
		recall.ArmDense: {{ID: "a", Rank: 0}, {ID: "b", Rank: 1}},
	})
	require.Len(t, got, 1)
	require.Equal(t, "b", got[0].ID)
}

func TestAgreementBetweenArmsOutranksEitherAlone(t *testing.T) {
	got := recall.FuseRRF(map[recall.Arm][]recall.Ranked{
		recall.ArmDense:   ranked("both", "dense-only"),
		recall.ArmLexical: ranked("lexical-only", "both"),
	})
	require.Equal(t, "both", got[0].ID)
	require.InDelta(t, 1.0/61.0+1.0/62.0, got[0].Score, 1e-12)
	require.Len(t, got[0].Contributions, 2)
}

func TestFusionIsDeterministicUnderTies(t *testing.T) {
	arms := map[recall.Arm][]recall.Ranked{
		recall.ArmDense:   ranked("z", "y", "x"),
		recall.ArmLexical: ranked("x", "y", "z"),
	}
	first := recall.FuseRRF(arms)
	for range 20 {
		require.Equal(t, first, recall.FuseRRF(arms),
			"a ranking that reorders between identical runs cannot be regression-tested")
	}
	// x and z each score 1/61 + 1/63; y scores 2/62, which is very slightly
	// lower. The tie between x and z is broken by identifier, and it breaks
	// the same way every time.
	require.InDelta(t, first[0].Score, first[1].Score, 1e-15, "x and z tie")
	require.Equal(t, "x", first[0].ID)
	require.Equal(t, "z", first[1].ID)
	require.Equal(t, "y", first[2].ID)
	require.Less(t, first[2].Score, first[0].Score,
		"agreeing on second place loses to disagreeing about first")
}

func TestContributionsAreInspectable(t *testing.T) {
	got := recall.FuseRRF(map[recall.Arm][]recall.Ranked{
		recall.ArmDense:   {{ID: "a", Rank: 3, Raw: -0.82}},
		recall.ArmLexical: {{ID: "a", Rank: 7, Raw: 0.19}},
	})
	require.Len(t, got, 1)
	require.Len(t, got[0].Contributions, 2)

	byArm := map[recall.Arm]recall.Contribution{}
	for _, c := range got[0].Contributions {
		byArm[c.Arm] = c
	}
	require.Equal(t, 3, byArm[recall.ArmDense].Rank)
	require.InDelta(t, -0.82, byArm[recall.ArmDense].Raw, 1e-12)
	require.InDelta(t, 1.0/63.0, byArm[recall.ArmDense].Score, 1e-12)
	require.Equal(t, 7, byArm[recall.ArmLexical].Rank)
	require.InDelta(t, 1.0/67.0, byArm[recall.ArmLexical].Score, 1e-12)
	require.InDelta(t, 1.0/63.0+1.0/67.0, got[0].Score, 1e-12)
}

func TestEqualDepth(t *testing.T) {
	require.True(t, recall.EqualDepth(map[recall.Arm][]recall.Ranked{
		recall.ArmDense:   ranked("a", "b", "c"),
		recall.ArmLexical: ranked("d", "e", "f"),
	}, 3))
	require.True(t, recall.EqualDepth(map[recall.Arm][]recall.Ranked{
		recall.ArmDense:   ranked("a"),
		recall.ArmLexical: nil,
	}, 3), "a short arm is fine when the corpus ran out")
	require.False(t, recall.EqualDepth(map[recall.Arm][]recall.Ranked{
		recall.ArmDense: ranked("a", "b", "c", "d"),
	}, 3))
}

func TestSingleArmShareAlarmsOnOneArmSearch(t *testing.T) {
	onlyDense := recall.FuseRRF(map[recall.Arm][]recall.Ranked{
		recall.ArmDense:   ranked("a", "b", "c", "d"),
		recall.ArmLexical: nil,
	})
	arm, share := recall.SingleArmShare(onlyDense, 4)
	require.Equal(t, recall.ArmDense, arm)
	require.InDelta(t, 1.0, share, 1e-12)

	balanced := recall.FuseRRF(map[recall.Arm][]recall.Ranked{
		recall.ArmDense:   ranked("a", "b"),
		recall.ArmLexical: ranked("a", "b"),
	})
	_, share = recall.SingleArmShare(balanced, 2)
	require.InDelta(t, 0.0, share, 1e-12)
}

func TestEmptyInputIsEmptyOutput(t *testing.T) {
	require.Empty(t, recall.FuseRRF(nil))
	require.Empty(t, recall.FuseRRF(map[recall.Arm][]recall.Ranked{}))
	arm, share := recall.SingleArmShare(nil, 10)
	require.Empty(t, arm)
	require.Zero(t, share)
}
