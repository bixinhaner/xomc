<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.addButtonDiv{
		position: absolute;
		right: 20px;
		top: 10px;
		width: 36px;
	}
	.tableDiv{
		height: 20px
	}
	.el-card__body{
		padding:0px;
	}
	.slide-content{
		border:none;
	}
</style>

<div class="panelDefault" id="modify_device_record">
	<el-ctable ref="recordList" id="cpeModifyPwdProgress" time="6" :rownumber="true" :height="'99%'" :url="tbURL" :pagination="true" :query-params="queryParam">
		<!-- 查询域 -->
		<template slot="toolbar">
			<div class="queryGroup" style="margin-left: 15px;">
				<input placeholder="<%=rb.getString("CPEXuLieHao")%> / <%=rb.getString("CPEName")%>" v-model="form.searchText"/>
				<b class="el-icon el-icon-common-search" @click="query"></b>
			</div>
		</template>
		<!-- 表格列 -->
		<el-table-column label='<%=rb.getString("CPEXuLieHao")%>' width="200"  prop="serialNumber"></el-table-column>
		<el-table-column label='<%=rb.getString("CPEName")%>' width="200"  prop="cpeName"></el-table-column>
		<el-table-column label='<%=rb.getString("ZhuangTai")%>' width="150" prop="status">
            <template slot-scope="scope">
               <div v-html="resultTableStatus(scope.row.status)"></div>
            </template>
        </el-table-column>
		<el-table-column label='<%=rb.getString("JieGuo")%>' prop="result">
            <template slot-scope="scope">
                <span v-if="scope.row.result == '1'"><%=rb.getString("ChengGong")%></span>
                <span v-if="scope.row.result == '2'"><%=rb.getString("ZhongZhi")%></span>
                <span v-if="scope.row.result == '3'"><%=rb.getString("ShiBai")%></span>
            </template>
        </el-table-column>
		<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failureReason"></el-table-column>
		<el-table-column label='<%=rb.getString("ShiJian")%>' prop="time"></el-table-column>
	</el-ctable>
</div>

<script type="text/javascript">
	new Vue({
		el: '#modify_device_record',
		data(){
			
			return {
				tbURL: '',
				exportURL: '${ctx}/task/cpe/changepwd/exportCPEChangePpasswordProgResultToCSV.action',
				queryParam: {
					searchText: '',
					taskId: '',
					timeZone: timeZone
				},
				form: {
					searchText: ''
				}
    	    }
		},
		methods: {
			init(id){
				var vm = this;
				vm.queryParam.taskId = id;
				vm.tbURL = '${ctx}/task/cpe/changepwd/getCPEChangePasswordProgress.action';
			},
			query(){
				var vm = this;
				Object.assign(vm.queryParam, vm.form);
				vm.$refs.recordList.refresh();
			},
			exportRecord() {
				var vm = this;
				
				vm.createForm(vm.exportURL, vm.queryParam);
			},
			createForm(url,param) {
				var body = document.querySelector('body'),
					form = document.createElement('form'),
					params = param || {};
				
				form.style.display = 'none';
				form.action = url;
				form.method = 'post';
				
				if(params) {
					params.token = omctoken;
					for(var key in params) {
						var input = document.createElement('input');
						input.value = params[key];
						input.setAttribute('name',key);
						form.appendChild(input);
					}
				}
				
				body.appendChild(form);
				form.submit();
				form.remove();
			}
		},
		mounted(){
			eventBus.$off('action-record').$on('action-record',this.init);
			eventBus.$off('action-export').$on('action-export',this.exportRecord);
		}
	});
</script>