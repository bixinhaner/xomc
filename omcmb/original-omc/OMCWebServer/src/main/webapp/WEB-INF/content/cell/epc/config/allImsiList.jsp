<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
    .active-status-div {
        display: flex;
        align-items: center
    }
    .active-status-div > i {
        margin-right: 5px;
        font-size: 20px;
    }
    .active-status-div .success-status::before {
        color: #67D972;
    }
    .el-icon-status-redirect:before{
     	color: #E88282;
     }
</style>

<div id="all_imsi_list">
    <el-ctable ref="imsi"
        row-key="imsi"
        :url="url"
        :query-params="queryParams">

        <template slot="toolbar">
            <el-query type="normal" placeholder="IMSI" @query="query"></el-query>
        </template>

        <el-table-column width="40">
            <template slot-scope="scope">
                <i class="el-icon el-icon-operation-delete" @click="deleteIMSI(scope.row)"></i>
            </template>
        </el-table-column>
        <el-table-column label="IMSI" prop="imsi"></el-table-column>
        <el-table-column label="<%=rb.getString("IMSIShenQinShiJian")%>" prop="requestTime"></el-table-column>
        <el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="assignStatus">
            <template slot-scope="scope">
                <div v-if="scope.row.assignStatus == '1'" class="active-status-div"><i class="el-icon el-icon-operation-active success-status"></i> <%=rb.getString("YiFenPei")%></div>
                <div v-if="scope.row.assignStatus == '2'" class="active-status-div"><i class="el-icon el-icon-operation-deactive"></i> <%=rb.getString("FenPeiShiBai")%></div>
                <div v-if="scope.row.assignStatus == '0'" class="active-status-div"><i class="el-icon el-icon-operation-discard1"></i> <%=rb.getString("YiShanChu")%></div>
            </template>
        </el-table-column>
        <el-table-column label="<%=rb.getString("IMSIJianQuanShiJian")%>" prop="responseTime"></el-table-column>
        <el-table-column label="<%=rb.getString("JiHuoZhuangTai")%>" prop="activeStatus">
            <template slot-scope="scope">
                <div v-if="scope.row.activeStatus == '0'" class="active-status-div"><i class="el-icon el-icon-status-de-active"></i> <%=rb.getString("WeiJiHuo")%></div>
                <div v-if="scope.row.activeStatus == '1'" class="active-status-div"><i class="el-icon el-icon-status-active1 success-status"></i> <%=rb.getString("JiHuo")%></div>
                <div v-if="scope.row.activeStatus == '2'" class="active-status-div"><i class="el-icon el-icon-status-redirect"></i> Redirect</div>
            </template>
        </el-table-column>
        <el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failureReason"></el-table-column>
        <el-table-column label="APN" prop="apnAssignStatus">
            <template slot-scope="scope">
                <div v-if="scope.row.apnAssignStatus == '0'" class="active-status-div"><i class="el-icon el-icon-status-vouchercard-available"></i> <%=rb.getString("WeiFenPei")%></div>
                <div v-if="scope.row.apnAssignStatus == '1'" class="active-status-div"><i class="el-icon el-icon-status-success success-status"></i> <%=rb.getString("YiFenPei")%></div>
                <div v-if="scope.row.apnAssignStatus == '2'" class="active-status-div"><i class="el-icon el-icon-status-failed"></i> <%=rb.getString("FenPeiShiBai")%></div>
            </template>
        </el-table-column>
        <el-table-column label="<%=rb.getString("APNShenQinShiJian")%>" prop="apnRequestTime"></el-table-column>
        <el-table-column label="<%=rb.getString("APNFenPeiShiJian")%>" prop="apnResponseTime"></el-table-column>
    </el-ctable>
</div>

<script>
    new Vue({
        el: '#all_imsi_list',
        data() {

            return {
                url: '',
                queryParams: {
                    searchText: '',
                    smallCellCode: '',
                    timeZone: timeZone
                }
            };
        },
        methods: {
            init(row) {
                var vm = this;

                vm.queryParams.smallCellCode = row.smallCellCode;
                vm.url = '${ctx}/cell/imsi/queryRequestImsiPageList.action';
            },
            query(txt) {
                var vm = this;

                vm.queryParams.searchText = txt;
            },
            deleteIMSI(row) {
                var vm = this,
                    url = '${ctx}/cell/imsi/sendImsiToEnb.action',
                    params = {
                        cmdType: 'clear',
                        smallCellCode: [vm.queryParams.smallCellCode],
                        imsi: []
                    };

                if(row.imsi) {
                    params.imsi.push(row.imsi);
                }

                params.smallCellCode = JSON.stringify(params.smallCellCode);
                params.imsi = JSON.stringify(params.imsi);

                vm.$confirm('<%=rb.getString("QueRenQingChuIMSI")%>','Confirm').then(function(r){
                    if(r) {
                        axios.post(url, stringify(params)).then(function(res){
                            var data = res.data;

                            if(data.success) {
                                vm.$message({
                                    type: 'success',
                                    message: '<%=rb.getString("ChengGong")%>'
                                });
                                vm.$refs.imsi.refresh();
                            }else {
                                vm.$message({
                                    type: 'error',
                                    message: data.message
                                });
                            }
                        });
                    }
                }).catch(function(){});
            }
        },
        mounted() {
            eventBus.$off('init-allimsi').$on('init-allimsi', this.init);
            document.body.click();
        }
    });
</script>