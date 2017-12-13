from cryptowatchbeat import BaseTest

import os


class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting Cryptowatchbeat normally
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/*"
        )

        cryptowatchbeat_proc = self.start_beat()
        self.wait_until( lambda: self.log_contains("cryptowatchbeat is running"))
        exit_code = cryptowatchbeat_proc.kill_and_wait()
        assert exit_code == 0
