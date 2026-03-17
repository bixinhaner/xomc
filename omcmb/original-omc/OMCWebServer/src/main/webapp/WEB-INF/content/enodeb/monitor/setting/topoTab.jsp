<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp"%>

<style>

</style>

<div id="topo_tab" style='height: 100%;overflow: auto;background: #fff;min-width: 900px;position: relative;'>
	<el-form style="height: 100%;">
		<el-collapse v-model="activeTabName">
			<el-collapse-item name="list">
                <template slot='title'>
					<p style="display:inline-block;margin-left:40px;">
						<span class="title-icon" style="vertical-align:sub"></span>
						<span style="font-size:14px;font-weight:bold">RRU</span>
					</p>
                    <div class="el-icon-common-refresh el-icon" @click='refreshTable'  style="position:absolute;right:50px;font-size:20px; top: 15px;"></div>
				</template>

				<div style="height:400px;width:95%;border:1px solid #F3F3F3">
					<el-ctable ref="list" :url="url" :query-params="queryParams" :pagination="false">
						<el-table-column label="SN" prop="SerialNumber"></el-table-column>
						<el-table-column label="IP" prop="IpAddress"></el-table-column>
						<el-table-column label="Sync Status" prop="TfcsSyncMgrState"></el-table-column>
						<el-table-column label="RF Status" prop="rfStatus">
                            <template slot-scope="scope">
                                <span v-if="[1,'1'].includes(scope.row.AdminState)">ON</span>
                                <span v-if="[0,'0'].includes(scope.row.AdminState)">OFF</span>
                            </template>
                        </el-table-column>
						<el-table-column label="Software Version" prop="SoftwareVersion"></el-table-column>
					</el-ctable>
				</div>

            </el-collapse-item>
        </el-collapse>
    </el-form>

</div>

<script>
    new Vue({
        el: '#topo_tab',
        data() {

            return {
                url: '${ctx}/cell/quicksettings/getRUToPo.action',
                activeTabName: 'list',
                dataList: [],
                queryParams: {
                    deviceCode: settingVue.selectedRow.small_cell_code
                }
            }
        },
        methods: {
            init() {
                var vm = this;

            },
            refreshTable(ev) {
                var vm = this,
                    params = {
                        //paramId: '31DC32429693829AC94A9CF75B88C908',
                        id: '',
					    smallCellCode: vm.queryParams.deviceCode
                    };

                axios.post('${ctx}/cell/quicksettings/sync.action',stringify(params)).then(function(res){
                    var data = res.data;

                    vm.$refs.list.refresh();
                })

                ev.stopPropagation();
                ev.preventDefault();
            }
        },
        mounted() {
            this.init();
        }
    })
</script>