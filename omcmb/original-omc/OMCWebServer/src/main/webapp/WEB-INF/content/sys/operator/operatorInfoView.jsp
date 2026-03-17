<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style type="text/css">
.headContent{
	font-size:14px;
	color:#333333;
	font-weight:700;
	display:flex;
	height:40px;
	align-items:center;
	padding-left:20px;
	padding-top: 10px;
	box-sizing:border-box;
}
.headContent img{
	margin-right:3px;
}
.panelDefault{
	display:flex;
	flex-direction:column;
	position: relative;
}
#viewoperatorSlide .slide-content{
	padding: 0px!important;
}
</style>

<div id="operatorViewPage" class="panelDefault">
	<div class="singleTitle" style="margin-left:10px;">
		<span v-show="queryParams.type == 'view'"><%=rb.getString("XinXi")%></span>
		<span v-show="queryParams.type == 'modify'"><%=rb.getString("XiuGai")%></span>
	</div>
	<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("FanHui")%>" style="right:40px;top:40px;">
		<span class="el-icon el-icon-circle-goback" @click="closeOperatorSlide"></span>
	</div>
	<div class="headContent">
		<img src="${ctx}/css/images/global/settingBetter.png"><span><%=rb.getString("JueSeJiLieBiao")%></span>
	</div>
	<el-ctable @selection-change="changeSelection" row-key="ROLE_ID" @load-success="loadSuccess" :default-checked="defaultChecked" ref="operatorViewTable" :url="operatorViewTableUrl" :query-params="queryParams" :height="height" pagination="true" :rownumber="rownumber">
		<el-table-column v-if="readOnlyFlag" type="selection" width="55" ></el-table-column>
		<el-table-column prop="role_name" label="<%=rb.getString("JueSeMingCheng")%>" ></el-table-column>
	</el-ctable>
</div>

<script>
new Vue({
	el:'#operatorViewPage',
	data(){
		return{
			readOnlyFlag:false,
			defaultChecked:[],
			operatorViewTableUrl:'',
			operator_code:'',
			height:'100%',
			pageSize:50,
			rownumber:true,
			oper_type:'',
			selectStr:'',
			queryParams : {
				'operator_code':'',
				'type':'',
			},
		}
	},
/* 	computed:{
		selectStr:function(){
			var vm = this;
			return vm.checkArr.join(",");
		}
	}, */
	methods:{
		viewInfo(operatprCode,type){
			var vm = this;
			vm.queryParams.operator_code = operatprCode;
			vm.queryParams.type = type;
			vm.readOnlyFlag = (type == 'view' ? false : true);
			vm.operatorViewTableUrl='${ctx}/system/operator/getSuperRoleListByOperatorCode.action';
		},
		loadSuccess(data){
			var vm = this;
			var filterArr = data.rows.filter(function(item){
				return item.check == "true"
			});
			vm.defaultChecked = filterArr.map(function(item){
				return item.ROLE_ID;
			})
		},
		changeSelection(selection){
			//this.checkArr =;
			var vm = this;
			var checkArr = selection.map(function(item){
				return item.ROLE_ID;
			})
			this.selectStr =  checkArr.join(",")
		},
		saveEdit(){
			var params = {};
			var vm = this;
			params.operator_code = this.queryParams.operator_code;
			params.role_ids = this.selectStr;
			axios.post('${ctx}/system/operator/save.action',stringify(params)).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			eventBus.$emit("close-slide");
	    			vm.$message({
	    				message:"<%=rb.getString("ChengGong")%>",
	    				type:'success'
	    			})
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	})
		},
		closeOperatorSlide(){
			eventBus.$emit("close-slide");
		}
	},
	mounted(){
		eventBus.$off("info-task").$on("info-task",this.viewInfo);
		eventBus.$off("save-edit-task").$on("save-edit-task",this.saveEdit);
		
	}
})

</script>