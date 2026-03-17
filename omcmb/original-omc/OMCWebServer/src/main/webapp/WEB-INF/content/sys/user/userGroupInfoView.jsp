<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#viewGroupInfo label{
		font-size:13px;
	}
</style>
<div id='viewGroupInfo'>
	<div style='margin-left:30px;margin-top:20px;'>
		<label><%=rb.getString("YiXuanJueSe")%></label>
		<div style='width:95%;height:300px;border:1px solid #DEDFE6;margin-top:5px;'>
			<el-ctable ref="groupInfoTable" :id="'groupInfoTable'" :url='roleUrlView' :width='width' :height="height"  :query-params="roleParams" page-size="20" pagination="true" row-key="role_id">
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input v-model='roleForm.role_name' @keyup.enter.native="queryRole" class='pairgrid-query' placeholder='<%=rb.getString("JueSeMingCheng")%>'></el-input>
						<i @click='queryRole' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column prop='role_name' label='<%=rb.getString("JueSeMingCheng")%>' width='200'></el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>'></el-table-column>
			</el-ctable>
		</div>
	</div>
	<div style='margin-left:30px;margin-top:30px;' v-if="cloudFlag =='false'">
		<label><%=rb.getString("YiXuanYongHu")%></label>
		<div style='width:95%;height:300px;border:1px solid #DEDFE6;margin-top:5px;'>
			<el-ctable :url='userUrl' :width='width' :height="height"  :query-params="userParams" page-size="20" pagination="true" row-key="user_id">
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input v-model='userForm.user_name' @keyup.enter.native="queryUser" class='pairgrid-query' placeholder='<%=rb.getString("YongHuMingCheng")%>'></el-input>
						<i @click='queryUser' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column prop='user_code' label='<%=rb.getString("YongHuMingCheng")%>' width='200'></el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("ZuMing")%>'></el-table-column>
			</el-ctable>
		</div>
	</div>
	<div style='margin-left:30px;margin-top:30px;width:95%'>
		<label style='float:left;width:90px;text-align:left;font-size:14px;color:#999'><%=rb.getString("MiaoShu")%>&nbsp;:</label><span style='color:#333;display:inline-block;width:89%;word-break:break-all;'>{{groupDesc}}</span>
	</div>
</div>
<script>
	new Vue({
		el:'#viewGroupInfo',
		data(){
			return{
				cloudFlag:isCloudCore,
				url:'',
				width:'100%',
				height:'100%',
				groupInfoParams:{
					
				},
				roleUrlView:'${ctx}/sys/usergroup/getRoleList.action',
				roleParams:{
					group_id:userVue.groupRowData.group_id,
					operator_code:operator_code,
					type:'view',
					role_name:''
				},
				roleForm:{
					role_name:''
				},
				userUrl:'${ctx}/sys/usergroup/getUserListForGroup.action',
				userParams:{
					group_id:userVue.groupRowData.group_id,
					operator_code:operator_code,
					type:'view',
					user_name:''
				},
				userForm:{
					user_name:''
				},
				groupDesc:''
			}
		},
		methods:{
			init(){
				this.groupDesc = userVue.groupRowData.group_desc
			},
			//角色列表模糊查询
			queryRole(){
				var vm = this;
				Object.assign(vm.roleParams, vm.roleForm);
			},
			//用户列表模糊查询
			queryUser(){
				var vm = this;
				Object.assign(vm.userParams, vm.userForm);
			}
		},
		mounted(){
			this.init();
		}
	})
</script>