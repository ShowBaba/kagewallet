package jobs

// func ReconcilePendingWithdrawals() {
// 	pendingWithdrawals, err := w.WithdrawalRepo.GetPendingWithdrawals()
// 	if err != nil {
// 		log.Error("Failed to fetch pending withdrawals", zap.Error(err))
// 		return
// 	}
//
// 	for _, withdrawal := range pendingWithdrawals {
// 		status, err := w.MonnifyService.GetWithdrawalStatus(withdrawal.TransactionID)
// 		if err != nil {
// 			log.Error("Failed to fetch withdrawal status", zap.Error(err))
// 			continue
// 		}
//
// 		if status != "pending" {
// 			err := w.WithdrawalRepo.UpdateWithdrawalStatus(withdrawal.ID, status)
// 			if err != nil {
// 				log.Error("Failed to update withdrawal status", zap.Error(err))
// 			}
// 		}
// 	}
// }
