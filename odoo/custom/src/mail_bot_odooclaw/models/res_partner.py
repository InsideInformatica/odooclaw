from odoo import models


class ResPartner(models.Model):
    _inherit = "res.partner"

    def _compute_im_status(self):
        """
        Override to expose the OdooClaw user as a bot in Discuss.
        """
        odooclaw_user = self.env.ref(
            "mail_bot_odooclaw.odooclaw_bot", raise_if_not_found=False
        )
        if odooclaw_user:
            odooclaw_partner = odooclaw_user.partner_id
            if odooclaw_partner in self:
                odooclaw_partner.im_status = "bot"
                if "offline_since" in odooclaw_partner._fields:
                    odooclaw_partner.offline_since = False

        to_process = self.filtered(
            lambda r: not odooclaw_user or r != odooclaw_user.partner_id
        )
        if not to_process:
            return
        return super(ResPartner, to_process)._compute_im_status()
